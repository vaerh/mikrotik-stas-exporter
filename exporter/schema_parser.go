package exporter

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

type ResSchema struct {
	ResourceName     string            `yaml:"resource_name"`            // Static resource name
	ResourcePath     string            `yaml:"resource_path"`            // Resource path
	ResourceFilter   map[string]string `yaml:"filter,omitempty"`         // Filter to find resource instances with the required parameters
	NameField        string            `yaml:"name_field"`               // Resource instance name
	StatFields       []string          `yaml:"stat_fields,omitempty"`    // Statistics fields
	StatFieldsRegExp string            `yaml:"stat_fields_re,omitempty"` // Regular expression for searching statistics fields
	SkipFields       []string          `yaml:"skip_fields,omitempty"`    // Forced skip fields

	resourceFilter *regexp.Regexp
	statFields     map[string]struct{}
	skipFields     map[string]struct{}
}

func (s *ResSchema) GetResourceFilter() *regexp.Regexp {
	return s.resourceFilter
}

func (s *ResSchema) GetStatFields() map[string]struct{} {
	return s.statFields
}

func (s *ResSchema) GetSkipFields() map[string]struct{} {
	return s.skipFields
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
		return nil, err
	}

	res.statFields = make(map[string]struct{})
	for _, key := range res.StatFields {
		res.statFields[key] = struct{}{}
	}

	res.skipFields = make(map[string]struct{})
	for _, key := range res.SkipFields {
		res.skipFields[key] = struct{}{}
	}

	// Validation
	var diags []string

	if res.StatFieldsRegExp != "" {
		res.resourceFilter, err = regexp.Compile(res.StatFieldsRegExp)
		if err != nil {
			diags = append(diags, schemaFileName+": 'filter' "+err.Error())
		}
	}

	if res.ResourceName == "" {
		diags = append(diags, schemaFileName+": 'resource_name' must be filled in")
	}

	if res.ResourcePath == "" {
		diags = append(diags, schemaFileName+": 'resource_path' must be filled in")
	}

	if res.NameField == "" {
		diags = append(diags, schemaFileName+": 'name_field' must be filled in")
	}

	if len(res.StatFields) == 0 && res.StatFieldsRegExp == "" {
		diags = append(diags, schemaFileName+": One of the fields 'stat_fields' or 'stat_fields_re' must be filled in")
	}

	if len(diags) != 0 {
		return nil, errors.New(strings.Join(diags, "\n"))
	}

	return &res, nil
}
