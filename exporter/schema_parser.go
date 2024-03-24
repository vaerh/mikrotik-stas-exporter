package exporter

import (
	"fmt"
	"os"

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

	return &res, nil
}
