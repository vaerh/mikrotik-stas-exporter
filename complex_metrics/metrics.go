package complexmetrics

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

const DefaultMetricsCollectionInterval = 30 * time.Second

type Metric interface {
	Register(ctx context.Context, constLabels prometheus.Labels, reg prometheus.Registerer)
	StartCollecting(ctx context.Context) error
	GetCollectInterval() time.Duration
	SetCollectInterval(t time.Duration)
}

var ComplexMetrics = ComplexMetricsType{}

type ComplexMetricsType struct {
	Metrics []Metric
}

func (m *ComplexMetricsType) AddMetric(metric Metric) {
	m.Metrics = append(m.Metrics, metric)
}

func (m *ComplexMetricsType) Get() []Metric {
	return m.Metrics
}

type CollectFunc func(context.Context) error

func startCollecting(ctx context.Context, metric Metric, collectFunc CollectFunc) error {
	timer := time.NewTicker(metric.GetCollectInterval())

	firstRun := make(chan struct{}, 1)
	firstRun <- struct{}{}

	for done := false; !done; {
		select {
		case <-firstRun:
			if err := collectFunc(ctx); err != nil {
				return fmt.Errorf("exporting metrics: %w", err)
			}
		case <-timer.C:
			if err := collectFunc(ctx); err != nil {
				return fmt.Errorf("exporting metrics: %w", err)
			}
		case <-ctx.Done():
			zerolog.Ctx(ctx).Debug().Msg("terminating exporter")
			done = true
		}
	}

	return nil
}
