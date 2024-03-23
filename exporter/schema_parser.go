package exporter

import (
	"fmt"
	prom "github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v2"
	"os"
)

type ResSchema struct {
	// PromMetricName Name of the metric to be created
	PromMetricName string `yaml:"prom_metric_name"`
	// PromConstLabels Map of labels and label values that are constant for the metric
	PromConstLabels prom.Labels `yaml:"prom_const_labels"`
	// PromMetricHelp help description of the metric
	PromMetricHelp string `yaml:"prom_metric_help,omitempty"`

	// ResourcePath Resource path in routeros
	ResourcePath string `yaml:"resource_path"`
	// ResourceFilter Filter executed on find to select interfaces (optional)
	ResourceFilter map[string]string `yaml:"resource_filter,omitempty"`
	// LabelNameFields  Set of labels to the metrics, for each label, the field where the value for the metric to be extracted
	LabelNameFields map[string]string `yaml:"label_name_fields,omitempty"`
	// MetricValueField Regular expression for searching statistics fields
	MetricValueField string `yaml:"metric_value_field,omitempty"`
}

func SchemaParser(schemaFileName string) (*ResSchema, error) {
	if _, err := os.Stat(schemaFileName); err != nil {
		return nil, fmt.Errorf("failed to read resource schema file '%v', %v", schemaFileName, err)
	}

	bytes, err := os.ReadFile(schemaFileName)
	if err != nil {
		return nil, err
	}

	var res ResSchema

	if err := yaml.Unmarshal(bytes, &res); err != nil {
		return nil, fmt.Errorf("unmarshalling schema on file '%s': %w", schemaFileName, err)
	}

	return &res, nil
}
