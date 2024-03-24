package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"

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

	http.Handle("/metrics", promhttp.Handler())

	go func() {
		if err = http.ListenAndServe(":8080", nil); err != nil {
			logger.Fatal().Err(err).Msg("")
		}
	}()

	conf := &mikrotik.Config{
		Insecure: true,
		HostURL:  os.Getenv("HOSTURL"),
		Username: os.Getenv("USERNAME"),
		Password: os.Getenv("PASSWORD"),
	}

	client, err := mikrotik.NewClient(ctx, conf)
	if err != nil {
		logger.Fatal().Err(err).Msg("creating mikrotik client")
	}
	ctx = client.WithContext(ctx)

	signalChan := make(chan os.Signal, 10)
	signal.Notify(signalChan, os.Interrupt, os.Kill)

	wg := sync.WaitGroup{}
	ctx, cancelFn := context.WithCancel(ctx)

	globalReg := prometheus.NewRegistry()

	for _, s := range schemas {
		wg.Add(1)

		workerReg := prometheus.NewRegistry()
		globalReg.Register(workerReg)

		go func() {
			defer globalReg.Unregister(workerReg)

			rExporter := exporter.NewResourceExporter(ctx, &s, workerReg)
			if err := rExporter.ExportMetrics(ctx); err != nil {
				logger.Err(err).Msg("exporting metrics")
			}

			wg.Done()
		}()
	}

	for done := false; !done; {
		select {
		case <-signalChan:
			cancelFn()
			done = true
		}
	}

	log.Printf("waiting for exporters")
	wg.Wait()
}
