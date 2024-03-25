package exporter

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
	"github.com/vaerh/mikrotik-prom-exporter/mikrotik"
)

type ResourceExporter struct {
	ctx         context.Context
	schema      *ResourceSchema
	promMertics map[string]any
}

func NewResourceExporter(ctx context.Context, schema *ResourceSchema, reg *prom.Registry) *ResourceExporter {
	var exporter = &ResourceExporter{
		ctx:         ctx,
		schema:      schema,
		promMertics: make(map[string]any),
	}

	for _, metric := range schema.Metrics {
		switch metric.PromMetricType {
		case CounterVec:
			counter := promauto.NewCounterVec(prom.CounterOpts{
				Namespace:   schema.PromNamespace,
				Subsystem:   schema.PromSubsystem,
				Name:        metric.PromMetricName,
				Help:        metric.PromMetricHelp,
				ConstLabels: metric.constLabels,
			}, metric.GetLabels())

			reg.MustRegister(counter)
			exporter.promMertics[metric.PromMetricName] = counter

		case GaugeVec:
			gauge := promauto.NewGaugeVec(prom.GaugeOpts{
				Namespace:   schema.PromNamespace,
				Subsystem:   schema.PromSubsystem,
				Name:        metric.PromMetricName,
				Help:        metric.PromMetricHelp,
				ConstLabels: metric.constLabels,
			}, metric.GetLabels())

			reg.MustRegister(gauge)
			exporter.promMertics[metric.PromMetricName] = gauge
		}

		// FIXME
		spew.Dump(metric.GetLabels())
	}

	return exporter
}

func (r *ResourceExporter) ExportMetrics(ctx context.Context) error {
	// FIXME
	timer := time.NewTicker(time.Second * 5)

	for done := false; !done; {
		select {
		case <-timer.C:
			if err := r.exportMetrics(ctx); err != nil {
				return fmt.Errorf("exporting metrics: %w", err)
			}
		case <-ctx.Done():
			zerolog.Ctx(ctx).Debug().Msg("terminating exporter")
			done = true
		}
	}

	return nil
}

func (r *ResourceExporter) exportMetrics(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Msg("exporting resources")

	resource, err := r.ReadResource()
	if err != nil {
		return fmt.Errorf("reading resource: %w", err)
	}

	for _, instanceJSON := range resource {
		// collect metrics & labels
		for _, metric := range r.schema.Metrics {
			var res any
			var err error
			// Parse value
			switch strings.ToLower(metric.MtFieldType) {
			case Float64:
				res, err = strconv.ParseFloat(instanceJSON[metric.MtFieldName], 64)
				if err != nil {
					logger.Warn().Err(err).Msg("extracting value from resource")
					continue
				}
			}

			var labels = make(prom.Labels, len(metric.labels))
			for labelName, mtFieldName := range metric.labels {
				labels[labelName] = instanceJSON[mtFieldName]
			}

			switch m := r.promMertics[metric.PromMetricName].(type) {
			case *prom.CounterVec:
				m.With(labels).Add(res.(float64))
			case *prom.GaugeVec:
				m.With(labels).Add(res.(float64))
			}
		}
	}
	return err
}

func (r *ResourceExporter) ReadResource() ([]mikrotik.MikrotikItem, error) {
	if len(r.schema.ResourceFilter) == 0 {
		return mikrotik.Read(r.schema.MikrotikResourcePath, mikrotik.Ctx(r.ctx))
	}
	var filter []string
	for k, v := range r.schema.ResourceFilter {
		filter = append(filter, k+"="+v)
	}
	return mikrotik.ReadFiltered(filter, r.schema.MikrotikResourcePath, mikrotik.Ctx(r.ctx))
}
