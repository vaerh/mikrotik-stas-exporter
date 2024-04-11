package main

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"

	"github.com/rs/zerolog"
	complexmetrics "github.com/vaerh/mikrotik-prom-exporter/complex_metrics"
	"github.com/vaerh/mikrotik-prom-exporter/exporter"
	"github.com/vaerh/mikrotik-prom-exporter/mikrotik"

	_ "github.com/vaerh/mikrotik-prom-exporter/complex_metrics"
)

var (
	ctx    = context.Background()
	logger = zerolog.New(os.Stdout)
	// maxConcurrentWorkers = 10
)

func init() {
	ctx = logger.WithContext(ctx) // Attach the Logger to the context.Context
}

func main() {
	schemas, err := exporter.LoadResSchemas(ctx, "resources")
	if err != nil {
		logger.Fatal().Err(err).Msg("")
	}

	conf := &mikrotik.Config{
		Insecure: true,
		HostURL:  os.Getenv("HOSTURL"),
		Username: os.Getenv("USERNAME"),
		Password: os.Getenv("PASSWORD"),
	}
	globalVars := map[string]string{
		"HOSTURL":  conf.HostURL,
		"USERNAME": conf.Username,
		// FIXME
		"ALIAS": "Sample-Router",
	}

	routerUrl, err := url.Parse(conf.HostURL)
	if err != nil {
		log.Err(err).Msg("")
	} else {
		globalVars["HOSTNAME"] = routerUrl.Hostname()
	}

	ctx, cancelFn := context.WithCancel(ctx)

	client, err := mikrotik.NewClient(ctx, conf)
	if err != nil {
		logger.Fatal().Err(err).Msg("creating mikrotik client")
	}
	ctx = client.WithContext(ctx)

	signalChan := make(chan os.Signal, 10)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	wg := sync.WaitGroup{}
	// sem := semaphore.NewWeighted(maxConcurrentWorkers)

	globalReg := prometheus.NewRegistry()

	for _, m := range complexmetrics.ComplexMetrics {
		wg.Add(1)

		workerReg := prometheus.NewRegistry()
		globalReg.Register(workerReg)

		go func() {
			defer globalReg.Unregister(workerReg)

			m.Register(ctx, prometheus.Labels{}, workerReg)

			if err := m.StartCollecting(ctx); err != nil {
				logger.Err(err).Msg("exporting metrics")
			}

			wg.Done()
		}()
	}

	for _, s := range schemas {
		wg.Add(1)

		workerReg := prometheus.NewRegistry()
		globalReg.Register(workerReg)

		go func() {
			defer globalReg.Unregister(workerReg)

			rExporter := exporter.NewResourceExporter(ctx, &s, workerReg)
			rExporter.SetGlobalVars(globalVars)

			if err := rExporter.ExportMetrics(ctx); err != nil {
				logger.Err(err).Msg("exporting metrics")
			}

			wg.Done()
		}()
	}

	// http.Handle("/metrics", promhttp.Handler())
	http.Handle("/metrics", promhttp.HandlerFor(globalReg, promhttp.HandlerOpts{}))

	go func() {
		if err = http.ListenAndServe(":8080", nil); err != nil {
			logger.Fatal().Err(err).Msg("")
		}
	}()

	// for done := false; !done; {
	// 	select {
	// 	case <-signalChan:
	// 		cancelFn()
	// 		done = true
	// 	}
	// }

	<-signalChan
	cancelFn()

	log.Printf("waiting for exporters")
	wg.Wait()
}
