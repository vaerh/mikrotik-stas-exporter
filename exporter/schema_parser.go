package exporter

import (
	"fmt"
	"os"

	prom "github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v2"
)

//TODO It would be nice to have a validator for the scheme....

func SchemaParser(schemaFileName string) (*ResourceSchema, error) {
	if _, err := os.Stat(schemaFileName); err != nil {
		return nil, fmt.Errorf("failed to read resource schema file '%v', %v", schemaFileName, err)
	}

	bytes, err := os.ReadFile(schemaFileName)
	if err != nil {
		return nil, err
	}

	var res ResourceSchema

	if err := yaml.Unmarshal(bytes, &res); err != nil {
		return nil, fmt.Errorf("unmarshalling schema on file '%s': %w", schemaFileName, err)
	}

	// Add global labels
	var globalLabels, globalConstLabels = make(prom.Labels), make(prom.Labels)
	for key, val := range res.PromGlobalLabels {
		if len(val) > 1 && val[0] == '$' {
			globalLabels[key] = val[1:]
		} else {
			globalConstLabels[key] = val
		}
	}

	for i := range res.Metrics {
		// Add private labels
		res.Metrics[i].labels = make(prom.Labels, len(globalLabels))
		res.Metrics[i].constLabels = make(prom.Labels, len(globalConstLabels))

		for k, v := range globalLabels {
			res.Metrics[i].labels[k] = v
		}

		for k, v := range globalConstLabels {
			res.Metrics[i].constLabels[k] = v
		}

		for key, val := range res.Metrics[i].PromLabels {
			if len(val) > 1 && val[0] == '$' {
				res.Metrics[i].labels[key] = val[1:]
			} else {
				res.Metrics[i].constLabels[key] = val
			}
		}

		// Validate field_type
		if res.Metrics[i].MtFieldType == Const {
			res.Metrics[i].PromMetricOperation = OperSet
		}
	}

	return &res, nil
}
