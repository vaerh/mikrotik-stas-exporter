package main

import (
	"context"
	"errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v2"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"

	"github.com/rs/zerolog"
	complexmetrics "github.com/vaerh/mikrotik-prom-exporter/complex_metrics"
	"github.com/vaerh/mikrotik-prom-exporter/exporter"
	"github.com/vaerh/mikrotik-prom-exporter/mikrotik"

	_ "github.com/vaerh/mikrotik-prom-exporter/complex_metrics"
)

var (
	logger = zerolog.New(os.Stdout)
	// maxConcurrentWorkers = 10
)

const (
	hostUrlEnvVar      = "HOST_URL"
	hostUrlFlagName    = "host_url"
	usernameFlagName   = "username"
	usernameEnvVarName = "USERNAME"
	passwordFlagName   = "password"
	passwordEnvVarName = "PASSWORD"
)

func main() {

	app := &cli.App{
		Name:  "mikrotik-prom-exporter",
		Usage: "export metrics in prometheus format from a mikrotik device",
		Commands: []*cli.Command{
			{
				Name:         "export",
				Usage:        "",
				UsageText:    "",
				Description:  "",
				Category:     "",
				BashComplete: nil,
				Before: func(c *cli.Context) error {

					c.Context = logger.WithContext(c.Context)
					return nil
				},
				After:        nil,
				Action:       cli.ActionFunc(export),
				OnUsageError: nil,
				Subcommands:  nil,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     hostUrlFlagName,
						Usage:    "url of the router in format",
						EnvVars:  []string{hostUrlEnvVar},
						Required: true,
						Aliases:  []string{"r"},
					},
					&cli.StringFlag{
						Name:     usernameFlagName,
						Usage:    "username for router authentication",
						EnvVars:  []string{usernameEnvVarName},
						Required: true,
						Aliases:  []string{"u"},
					},
					&cli.StringFlag{
						Name:     passwordFlagName,
						Usage:    "password for router authentication",
						EnvVars:  []string{passwordEnvVarName},
						Required: true,
						Aliases:  []string{"p"},
					},
				},
				SkipFlagParsing:        false,
				HideHelp:               false,
				HideHelpCommand:        false,
				Hidden:                 false,
				UseShortOptionHandling: false,
				HelpName:               "",
				CustomHelpTemplate:     "",
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("failed to run export")
	}
}

func export(cliCtx *cli.Context) error {
	ctx := cliCtx.Context
	schemas, err := exporter.LoadResSchemas(ctx, "resources")
	if err != nil {
		logger.Fatal().Err(err).Msg("")
	}

	conf := mikrotikConfigFromFlags(cliCtx)

	globalVars := map[string]string{
		"HOSTURL":  conf.HostURL,
		"USERNAME": conf.Username,
		// FIXME
		"ALIAS": "Sample-Router",
	}

	_, err = url.Parse(conf.HostURL)
	if err != nil {
		log.Fatal().Err(err).Msg("parsing router host url")
	}

	ctx, cancelFn := context.WithCancel(ctx)

	client, err := mikrotik.NewClient(ctx, conf)
	if err != nil {
		logger.Fatal().Err(err).Msg("creating mikrotik client")
	}

	// start http service ASAP to be sure it actually is online
	globalReg := prometheus.NewRegistry()
	// http.Handle("/metrics", promhttp.Handler())
	http.Handle("/metrics", promhttp.HandlerFor(globalReg, promhttp.HandlerOpts{}))

	go func() {
		if err = http.ListenAndServe(":8080", nil); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				logger.Fatal().Err(err).Msg("listening and starting http server for metrics")
			}
		}
	}()

	ctx = client.WithContext(ctx)

	signalChan := make(chan os.Signal, 10)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	wg := sync.WaitGroup{}
	// sem := semaphore.NewWeighted(maxConcurrentWorkers)

	for _, m := range complexmetrics.ComplexMetrics.Metrics {
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
	return nil
}

func mikrotikConfigFromFlags(cliCtx *cli.Context) *mikrotik.Config {
	return &mikrotik.Config{
		Insecure: true,
		HostURL:  cliCtx.String(hostUrlFlagName),
		Username: cliCtx.String(usernameFlagName),
		Password: cliCtx.String(passwordFlagName),
	}
}
