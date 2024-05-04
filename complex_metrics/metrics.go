package complexmetrics

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

type Metric interface {
	Register(ctx context.Context, constLabels prometheus.Labels, reg prometheus.Registerer)
	StartCollecting(ctx context.Context) error
}

var ComplexMetrics = ComplexMetricsType{}

type ComplexMetricsType struct {
	Metrics []Metric
}

func (m *ComplexMetricsType) AddMetric(metric Metric) {
	m.Metrics = append(m.Metrics, metric)
}

// FIXME
var DataCollectionInterval = 5 * time.Second

type CollectFunc func(context.Context) error

func startCollecting(ctx context.Context, collectFunc CollectFunc) error {
	timer := time.NewTicker(DataCollectionInterval)

	for done := false; !done; {
		select {
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
