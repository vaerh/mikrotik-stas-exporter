package mikrotik

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
