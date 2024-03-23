package main

import (
	"context"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/rs/zerolog"
	"github.com/vaerh/mikrotik-prom-exporter/exporter"
	"github.com/vaerh/mikrotik-prom-exporter/mikrotik"
)

var (
	ctx    = context.Background()
	logger = zerolog.New(os.Stdout)
)

func init() {
	ctx = logger.WithContext(ctx) // Attach the Logger to the context.Context
}

func main() {

	schemas, err := exporter.LoadResSchemas(ctx, "resources")
	if err != nil {
		logger.Fatal().Err(err).Msg("")
	}

	for _, s := range *schemas {
		ReadResource(&s)
	}
}

func ReadResource(s *exporter.ResSchema) {
	conf := &mikrotik.Config{
		Insecure: true,
		HostURL:  os.Getenv("HOSTURL"),
		Username: os.Getenv("USERNAME"),
		Password: os.Getenv("PASSWORD"),
	}

	client, err := mikrotik.NewClient(ctx, conf)
	if err != nil {
		logger.Fatal().Err(err).Msg("")
	}

	var resource []mikrotik.MikrotikItem

	if len(s.ResourceFilter) == 0 {
		resource, err = mikrotik.Read("/interface", client)
	} else {
		var filter []string
		for k, v := range s.ResourceFilter {
			filter = append(filter, k+"="+v)
		}
		resource, err = mikrotik.ReadFiltered(filter, s.ResourcePath, client)
	}

	// res, err := mikrotik.Read("/system/identity", client)
	if err != nil {
		logger.Fatal().Err(err).Msg("")
	}

	var res = exporter.ResourceStats{Name: s.ResourceName}
	var reStat, statFields, skipFields = s.GetResourceFilter(), s.GetStatFields(), s.GetSkipFields()

	for _, instance := range resource {
		var is = exporter.InstanceStats{Stats: make(map[string]any)}

		for k, v := range instance {
			if k == s.NameField {
				is.Name = v
				continue
			}

			if _, ok := skipFields[k]; ok {
				continue
			}

			if _, ok := statFields[k]; !ok && !reStat.MatchString(k) {
				continue
			}

			is.Stats[k] = v
		}

		res.Instance = append(res.Instance, is)
	}

	spew.Dump(res)
}
