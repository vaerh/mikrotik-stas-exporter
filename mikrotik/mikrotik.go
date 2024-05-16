package mikrotik

import "context"

type TransportType int

// Using numbering from 1 to control type values.
const (
	TransportAPI TransportType = 1 + iota
	TransportREST
)

// MikrotikItem Contains only data.
type MikrotikItem map[string]string

func BoolFromMikrotikJSONToFloat(s string) float64 {
	if s == "true" || s == "yes" {
		return 1.0
	}
	return 0.0
}

func ReadResource(ctx context.Context, resourcePath string, resourceFilter map[string]string) ([]MikrotikItem, error) {
	if len(resourceFilter) == 0 {
		return Read(resourcePath, Ctx(ctx), nil)
	}
	var filter []string
	for k, v := range resourceFilter {
		filter = append(filter, k+"="+v)
	}
	return ReadFiltered(filter, resourcePath, Ctx(ctx), nil)
}
