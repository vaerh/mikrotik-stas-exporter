package metrics

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promauto"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/vaerh/mikrotik-prom-exporter/mikrotik"
)

type ResourceExporter struct {
	*anyMetric
	schema      *ResourceSchema
	promMetrics map[string]any
	globalVars  map[string]string
	labels      map[string]string
	reg         *prom.Registry
}

var _ Metric = (*ResourceExporter)(nil)

func NewResourceExporter(schema *ResourceSchema) *ResourceExporter {
	var exporter = &ResourceExporter{
		schema:      schema,
		promMetrics: make(map[string]any),
	}
	return exporter
}

func (r *ResourceExporter) Register(reg prom.Registerer) {
	for _, metric := range r.schema.Metrics {
		var cl = make(prom.Labels, len(metric.constLabels)+len(r.labels))
		for k, v := range metric.constLabels {
			cl[k] = v
		}
		for k, v := range r.labels {
			cl[k] = v
		}

		switch metric.PromMetricType {
		case CounterVec:
			counter := promauto.NewCounterVec(prom.CounterOpts{
				Namespace:   r.schema.PromNamespace,
				Subsystem:   r.schema.PromSubsystem,
				Name:        metric.PromMetricName,
				Help:        metric.PromMetricHelp,
				ConstLabels: cl,
			}, metric.GetLabels())

			r.reg.MustRegister(counter)
			r.promMetrics[metric.PromMetricName] = counter

		case GaugeVec:
			gauge := promauto.NewGaugeVec(prom.GaugeOpts{
				Namespace:   r.schema.PromNamespace,
				Subsystem:   r.schema.PromSubsystem,
				Name:        metric.PromMetricName,
				Help:        metric.PromMetricHelp,
				ConstLabels: cl,
			}, metric.GetLabels())

			r.reg.MustRegister(gauge)
			r.promMetrics[metric.PromMetricName] = gauge
		}

		// FIXME
		// spew.Dump(metric.GetLabels())
	}
}

func (r *ResourceExporter) Collect(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Msg("exporting resources")

	mikrotikResource, err := r.ReadResource(ctx)
	if err != nil {
		return fmt.Errorf("reading resource: %w", err)
	}

	// Zeroize
	for _, metric := range r.schema.Metrics {
		if metric.PromResetGaugeEveryTime {
			if g, ok := r.promMetrics[metric.PromMetricName].(*prom.GaugeVec); ok {
				g.Reset()
			}
		}
	}

	for _, instanceJSON := range mikrotikResource {
		// collect metrics & labels
		for _, metric := range r.schema.Metrics {
			var res any
			var err error
			// Parse value
			inVal := instanceJSON[metric.MtFieldName]
			switch strings.ToLower(metric.MtFieldType) {
			case Int:
				res, err = strconv.ParseFloat(inVal, 64)
				if err != nil {
					logger.Warn().Fields(map[string]any{metric.MtFieldName: inVal}).Err(err).Msg("extracting value from resource")
					continue
				}
			case Time:
				d, err := mikrotik.ParseDuration(inVal)
				if err != nil {
					logger.Warn().Fields(map[string]any{metric.MtFieldName: inVal}).Err(err).Msg("extracting value from resource")
					continue
				}
				res = d.Seconds()
			case Const:
				res = float64(1.0)
			case Bool:
				res = mikrotik.BoolFromMikrotikJSONToFloat(inVal)
			}

			var labels = make(prom.Labels, len(metric.labels))
			for labelName, mtFieldName := range metric.labels {
				labels[labelName] = instanceJSON[mtFieldName]
				if v, ok := r.globalVars[mtFieldName]; ok {
					labels[labelName] = v
				}
			}

			switch m := r.promMetrics[metric.PromMetricName].(type) {
			case *prom.CounterVec:
				if metric.PromMetricOperation == OperAdd {
					m.With(labels).Add(res.(float64))
				} else {
					m.With(labels).Inc()
				}
			case *prom.GaugeVec:
				switch metric.PromMetricOperation {
				case OperInc:
					m.With(labels).Inc()
				case OperDec:
					m.With(labels).Dec()
				case OperAdd:
					m.With(labels).Add(res.(float64))
				case OperSub:
					m.With(labels).Sub(res.(float64))
				case OperCurrTime:
					m.With(labels).SetToCurrentTime()
				case OperSet:
					fallthrough
				default:
					m.With(labels).Set(res.(float64))
				}
			}
		}
	}
	return err
}

func (r *ResourceExporter) ReadResource(ctx context.Context) ([]mikrotik.MikrotikItem, error) {
	if len(r.schema.ResourceFilter) == 0 {
		return mikrotik.Read(r.schema.MikrotikResourcePath, mikrotik.Ctx(ctx), nil)
	}
	var filter []string
	for k, v := range r.schema.ResourceFilter {
		filter = append(filter, k+"="+v)
	}
	return mikrotik.ReadFiltered(filter, r.schema.MikrotikResourcePath, mikrotik.Ctx(ctx), nil)
}
