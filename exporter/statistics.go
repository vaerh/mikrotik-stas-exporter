package exporter

type ResourceStats struct {
	Name     string
	Instance []InstanceStats
}

type InstanceStats struct {
	Name  string
	Stats map[string]any
}
