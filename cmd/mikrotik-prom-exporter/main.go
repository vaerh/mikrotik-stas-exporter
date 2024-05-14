package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/vaerh/mikrotik-prom-exporter/exporter"
	"github.com/vaerh/mikrotik-prom-exporter/mikrotik"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	complexmetrics "github.com/vaerh/mikrotik-prom-exporter/complex_metrics"
)

var (
	logger = zerolog.New(os.Stdout)
	// maxConcurrentWorkers = 10
)

var (
	flagHostURL = &cli.StringFlag{
		Name:     "hosturl",
		Usage:    "`URL` of the router in format",
		EnvVars:  []string{"HOSTURL"},
		Required: true,
		Aliases:  []string{"r"},
	}
	flagUsername = &cli.StringFlag{
		Name:     "username",
		Usage:    "`USERNAME` for router authentication",
		EnvVars:  []string{"USERNAME"},
		Required: true,
		Aliases:  []string{"u"},
	}
	flagPassword = &cli.StringFlag{
		Name:     "password",
		Usage:    "`PASSWORD` for router authentication",
		EnvVars:  []string{"PASSWORD"},
		Required: true,
		Aliases:  []string{"p"},
	}
	flagInsecure = &cli.BoolFlag{
		Name:    "insecure", // curl -k/--insecure
		Usage:   "don't check the server certificate",
		EnvVars: []string{"INSECURE"},
		Aliases: []string{"k"},
	}
	flagCaCert = &cli.StringFlag{
		Name:    "cacert", // curl --cacert
		Usage:   "certificate `FILE` to verify the router",
		EnvVars: []string{"CA_CERTIFICATE"},
	}
	flagRouterAlias = &cli.StringFlag{
		Name:        "alias",
		Usage:       "router `ALIAS` to display in metrics labels",
		EnvVars:     []string{"ROUTER_ALIAS"},
		Value:       "Sample-Router",
		DefaultText: "Sample-Router",
	}
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
				After:        nil,
				Action:       cli.ActionFunc(export),
				OnUsageError: nil,
				Subcommands:  nil,
				Flags: []cli.Flag{
					flagHostURL,
					flagUsername,
					flagPassword,
					flagInsecure,
					flagCaCert,
					flagRouterAlias,
					&cli.IntFlag{
						Name:        "listen",
						Usage:       "mikrotik exporter `PORT`",
						Value:       9100,
						DefaultText: "9100",
						EnvVars:     []string{"LISTEN_PORT"},
						Action: func(ctx *cli.Context, v int) error {
							if v < 0 || v > 65535 {
								return fmt.Errorf("flag port value %v out of range[0-65535]", v)
							}
							return nil
						},
						Aliases: []string{"l"},
					},
					&cli.StringFlag{
						Name:        "interval",
						Usage:       "Positive `INTERVAL` of metrics collection https://pkg.go.dev/time#ParseDuration",
						Value:       "30s",
						DefaultText: "30s",
						EnvVars:     []string{"INTERVAL"},
						Action: func(ctx *cli.Context, v string) error {
							t, err := time.ParseDuration(v)
							if err != nil {
								return fmt.Errorf("metrics collection interval parsing error, %v", err)
							}
							if t < 5*time.Second {
								return fmt.Errorf("metrics collection interval '%v' must be greater than or equal to 5 seconds", v)
							}
							return nil
						},
						Aliases: []string{"i"},
					},
					&cli.StringFlag{
						Name:        "resources",
						Usage:       "`DIR`ECTORY with metrics schemas",
						Value:       "resources",
						DefaultText: "resources",
						EnvVars:     []string{"RESOURCES_DIR"},
						Action: func(ctx *cli.Context, v string) error {
							_, err := os.Stat(v)
							if err != nil {
								return fmt.Errorf("error checking directory with metrics schemas, %v", err)
							}
							return nil
						},
					},
					&cli.StringFlag{
						Name:        "loglevel",
						Usage:       "Log `LEVEL`",
						Value:       "warn",
						DefaultText: "warn",
						EnvVars:     []string{"LOG_LEVEL"},
						Action: func(ctx *cli.Context, v string) error {
							lvl, err := zerolog.ParseLevel(v)
							if err != nil {
								return fmt.Errorf("error parsing log level, %v", err)
							}
							ctx.Context = logger.Level(lvl).WithContext(ctx.Context)
							return nil
						},
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

	conf := &mikrotik.Config{
		Insecure:      flagInsecure.Get(cliCtx),
		CaCertificate: flagCaCert.Get(cliCtx),
		HostURL:       flagHostURL.Get(cliCtx),
		Username:      flagUsername.Get(cliCtx),
		Password:      flagPassword.Get(cliCtx),
	}

	u, err := url.Parse(flagHostURL.Get(cliCtx))
	if err != nil {
		log.Fatal().Err(err).Msg("parsing router host url")
	}

	globalVars := map[string]string{
		"HOSTURL":  u.Host,
		"USERNAME": flagUsername.Get(cliCtx),
		"ALIAS":    flagRouterAlias.Get(cliCtx),
	}

	ctx, cancelFn := context.WithCancel(ctx)

	client, err := mikrotik.NewClient(ctx, conf)
	if err != nil {
		logger.Fatal().Err(err).Msg("creating mikrotik client")
	}

	res, err := client.SendRequest(mikrotik.CrudRead, &mikrotik.URL{Path: "/system/identity"}, nil)
	if err != nil {
		logger.Err(err).Msg("read router identity")
	} else {
		if len(res) > 0 {
			globalVars["ROUTER_ID"] = res[0]["name"]
		}
	}

	// start http service ASAP to be sure it actually is online
	globalReg := prometheus.NewRegistry()
	// http.Handle("/metrics", promhttp.Handler())
	http.Handle("/metrics", promhttp.HandlerFor(globalReg, promhttp.HandlerOpts{}))

	go func() {
		if err = http.ListenAndServe(fmt.Sprintf(":%d", cliCtx.Int("listen")), nil); err != nil {
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

	metricsCollectionInterval, _ := time.ParseDuration(cliCtx.String("interval"))

	for _, m := range complexmetrics.ComplexMetrics.Get() {
		wg.Add(1)

		workerReg := prometheus.NewRegistry()
		globalReg.MustRegister(workerReg)

		go func() {
			defer globalReg.Unregister(workerReg)

			m.Register(ctx, prometheus.Labels{
				"routerboard_address": globalVars["HOSTURL"],
				"routerboard_id":      globalVars["ROUTER_ID"],
				"routerboard_alias":   globalVars["ALIAS"]},
				workerReg)
			m.SetCollectInterval(metricsCollectionInterval)

			if err := m.StartCollecting(ctx); err != nil {
				logger.Err(err).Msg("exporting metrics")
			}

			wg.Done()
		}()
	}

	for _, s := range schemas {
		wg.Add(1)

		workerReg := prometheus.NewRegistry()
		globalReg.MustRegister(workerReg)

		go func() {
			defer globalReg.Unregister(workerReg)

			rExporter := exporter.NewResourceExporter(ctx, &s, prometheus.Labels{
				"routerboard_address": globalVars["HOSTURL"],
				"routerboard_id":      globalVars["ROUTER_ID"],
				"routerboard_alias":   globalVars["ALIAS"]},
				workerReg)
			rExporter.SetGlobalVars(globalVars)
			rExporter.SetCollectInterval(metricsCollectionInterval)

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
