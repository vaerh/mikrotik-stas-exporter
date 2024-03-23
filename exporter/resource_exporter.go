package exporter

import (
	"context"
	"fmt"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
	"github.com/vaerh/mikrotik-prom-exporter/mikrotik"
	"strconv"
	"time"
)

type ResourceExporter struct {
	counterVec *prom.CounterVec
	schema     ResSchema
	client     mikrotik.Client
}

func NewResourceExporter(schema ResSchema, client mikrotik.Client) *ResourceExporter {
	labels := make([]string, 0, len(schema.LabelNameFields))
	for key, _ := range schema.LabelNameFields {
		labels = append(labels, key)
	}
	counter := promauto.NewCounterVec(prom.CounterOpts{
		Namespace:   "",
		Subsystem:   "",
		Name:        schema.PromMetricName,
		Help:        schema.PromMetricHelp,
		ConstLabels: schema.PromConstLabels,
	}, labels)

	return &ResourceExporter{
		counterVec: counter,
		client:     client,
		schema:     schema,
	}
}

func (r *ResourceExporter) ExportMetrics(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	timer := time.NewTicker(time.Second * 30)
	for done := false; !done; {
		select {
		case <-timer.C:
			if err := r.exportMetrics(ctx); err != nil {
				return fmt.Errorf("exporting metrics: %w", err)
			}
		case <-ctx.Done():
			logger.Debug().Msg("terminating exporter")
			done = true
		}
	}
	return nil
}

func (r *ResourceExporter) exportMetrics(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Msg("exporting resources")

	resources, err := r.ReadResource()
	if err != nil {
		return fmt.Errorf("reading resource: %w", err)
	}

	for _, instance := range resources {
		labels, err := extractLabelsFromResource(instance, r.schema.LabelNameFields)
		if err != nil {
			logger.Warn().Err(err).Msg("extracting labels from resource")
			continue
		}
		metricValue, err := extractValueFromResource(instance, r.schema.MetricValueField)
		if err != nil {
			logger.Warn().Err(err).Msg("extracting value from resource")
			continue
		}
		r.counterVec.With(labels).Add(metricValue)
	}
	return err
}

func extractValueFromResource(instance mikrotik.MikrotikItem, field string) (float64, error) {
	value := instance[field]
	return strconv.ParseFloat(value, 64)
}

func (r *ResourceExporter) ReadResource() ([]mikrotik.MikrotikItem, error) {
	if len(r.schema.ResourceFilter) == 0 {
		return mikrotik.Read(r.schema.ResourcePath, r.client)
	}
	var filter []string
	for k, v := range r.schema.ResourceFilter {
		filter = append(filter, k+"="+v)
	}
	return mikrotik.ReadFiltered(filter, r.schema.ResourcePath, r.client)
}

func extractLabelsFromResource(instance mikrotik.MikrotikItem, labelNamesFields map[string]string) (labels prom.Labels, err error) {
	labels = make(prom.Labels, len(labelNamesFields))
	for labelName, labelField := range labelNamesFields {
		labelValue, ok := instance[labelField]
		if !ok {
			return nil, fmt.Errorf("unable to find:%s in resrouce: %v", labelField, instance)
		}
		labels[labelName] = labelValue
	}
	return labels, nil
}
