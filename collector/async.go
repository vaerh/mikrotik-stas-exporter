package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/vaerh/mikrotik-prom-exporter/metrics"
)

type metricsCollectorOptions struct {
	globalRegistry     *prometheus.Registry
	constLabels        prometheus.Labels
	collectionInterval time.Duration
	globalVars         map[string]string
}

type MetricsCollectorOptions func(*metricsCollectorOptions) error

func WithGlobalRegistry(registry *prometheus.Registry) MetricsCollectorOptions {
	return func(o *metricsCollectorOptions) error {
		o.globalRegistry = registry
		return nil
	}
}

func WithConstLabels(labels prometheus.Labels) MetricsCollectorOptions {
	return func(o *metricsCollectorOptions) error {
		o.constLabels = labels
		return nil
	}
}

func WithCollectionInterval(collectionInterval time.Duration) MetricsCollectorOptions {
	return func(o *metricsCollectorOptions) error {
		o.collectionInterval = collectionInterval
		return nil
	}
}

func WithGlobalVars(globalVars map[string]string) MetricsCollectorOptions {
	return func(o *metricsCollectorOptions) error {
		o.globalVars = globalVars
		return nil
	}
}

type AsyncCollector struct {
	*metricsCollectorOptions
}

type MetricAsyncCollector struct {
	MetricsCollectorOptions
}

func NewAsyncCollector(opts ...MetricsCollectorOptions) *AsyncCollector {
	o := &metricsCollectorOptions{}
	for _, opt := range opts {
		if err := opt(o); err != nil {
			panic(fmt.Errorf("creating async collector: %w", err))
		}
	}

	return &AsyncCollector{
		o,
	}
}

func (c *AsyncCollector) CollectMetrics(ctx context.Context, collectors ...metrics.Metric) {
	logger := zerolog.Ctx(ctx)

	wg := sync.WaitGroup{}
	for _, s := range collectors {
		wg.Add(1)

		workerReg := prometheus.NewRegistry()
		c.globalRegistry.MustRegister(workerReg)

		go func() {
			defer c.globalRegistry.Unregister(workerReg)

			s.SetConstLabels(c.constLabels)
			s.SetGlobalVars(c.globalVars)
			s.Register(workerReg)

			if err := startCollecting(ctx, c.collectionInterval, s); err != nil {
				logger.Err(err).Msg("exporting metrics")
			}

			wg.Done()
		}()
	}

	wg.Wait()
}

func startCollecting(ctx context.Context, interval time.Duration, metric metrics.Metric) error {
	timer := time.NewTicker(interval)

	firstRun := make(chan struct{}, 1)
	firstRun <- struct{}{}

	for done := false; !done; {
		select {
		case <-firstRun:
			if err := metric.Collect(ctx); err != nil {
				return fmt.Errorf("exporting metrics: %w", err)
			}
		case <-timer.C:
			if err := metric.Collect(ctx); err != nil {
				return fmt.Errorf("exporting metrics: %w", err)
			}
		case <-ctx.Done():
			zerolog.Ctx(ctx).Debug().Msg("terminating exporter")
			done = true
		}
	}
	return nil
}
