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
	Callback(ctx context.Context) error
}

var ComplexMetrics []Metric

// FIXME
var DataCollectionInterval = 5 * time.Second

func startCollecting(ctx context.Context, m Metric) error {
	timer := time.NewTicker(DataCollectionInterval)

	for done := false; !done; {
		select {
		case <-timer.C:
			if err := m.Callback(ctx); err != nil {
				return fmt.Errorf("exporting metrics: %w", err)
			}
		case <-ctx.Done():
			zerolog.Ctx(ctx).Debug().Msg("terminating exporter")
			done = true
		}
	}

	return nil
}
