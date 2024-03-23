package mikrotik

type TransportType int

// Using numbering from 1 to control type values.
const (
	TransportAPI TransportType = 1 + iota
	TransportREST
)

// MikrotikItem Contains only data.
type MikrotikItem map[string]string

func BoolFromMikrotikJSON(s string) bool {
	if s == "true" || s == "yes" {
		return true
	}
	return false
}
