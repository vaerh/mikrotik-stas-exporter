package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const DefaultMetricsCollectionInterval = 30 * time.Second

type Metric interface {
	Register(reg prometheus.Registerer)
	Collect(ctx context.Context) error
	SetConstLabels(labels prometheus.Labels)
	SetGlobalVars(map[string]string)
}

type anyMetric struct {
	globalLabels prometheus.Labels
	globalVars   map[string]string
}

func (m *anyMetric) SetGlobalVars(vars map[string]string) {
	m.globalVars = vars
}

func (m *anyMetric) SetConstLabels(labels prometheus.Labels) {
	m.globalLabels = labels
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
