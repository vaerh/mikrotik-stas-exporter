package complexmetrics

import (
	"context"
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
	"github.com/vaerh/mikrotik-prom-exporter/mikrotik"
)

func init() {
	ComplexMetrics = append(ComplexMetrics, &InterfaceRate{path: "/interface/ethernet"})
}

type InterfaceRate struct {
	path    string
	duplex  *prometheus.GaugeVec
	rate    *prometheus.GaugeVec
	sfpTemp *prometheus.GaugeVec
	status  *prometheus.GaugeVec
}

// Callback implements Metric.
func (ir *InterfaceRate) Callback(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Msg("exporting resources")

	// curl -s -k -X POST -H "content-type: application/json" "https://172.16.3.1/rest/interface/ethernet/monitor?once" --data '{"numbers":"ether1","once":""}'
	mikrotikResource, err := mikrotik.ReadResource(ctx, ir.path, nil)
	if err != nil {
		return fmt.Errorf("reading resource: %w", err)
	}

	for _, iface := range mikrotikResource {
		// Get status
		if id, ok := iface[".id"]; ok {
			res, err := mikrotik.Monitor(ir.path, mikrotik.Ctx(ctx), map[string]string{
				"numbers": id,
				"once":    "",
			})
			if err != nil {
				return fmt.Errorf("reading resource: %w", err)
			}
			if len(res) == 0 {
				logger.Warn().Msgf("monitor empty response: %v", id)
				continue
			}

			if res[0]["status"] == "link-ok" {
				ir.status.With(prometheus.Labels{"name": res[0]["name"]}).Set(1)
			} else {
				ir.status.With(prometheus.Labels{"name": res[0]["name"]}).Set(0)
			}

			switch res[0]["rate"] {
			case "10Mbps":
				ir.rate.With(prometheus.Labels{"name": res[0]["name"]}).Set(10)
			case "100Mbps":
				ir.rate.With(prometheus.Labels{"name": res[0]["name"]}).Set(100)
			case "1Gbps":
				ir.rate.With(prometheus.Labels{"name": res[0]["name"]}).Set(1000)
			case "2.5Gbps":
				ir.rate.With(prometheus.Labels{"name": res[0]["name"]}).Set(2500)
			case "5Gbps":
				ir.rate.With(prometheus.Labels{"name": res[0]["name"]}).Set(5000)
			case "10Gbps":
				ir.rate.With(prometheus.Labels{"name": res[0]["name"]}).Set(10000)
			case "40Gbps":
				ir.rate.With(prometheus.Labels{"name": res[0]["name"]}).Set(40000)
			}

			if res[0]["full-duplex"] == "true" {
				ir.duplex.With(prometheus.Labels{"name": res[0]["name"]}).Set(1)
			} else {
				ir.duplex.With(prometheus.Labels{"name": res[0]["name"]}).Set(0)
			}

			if temp, ok := res[0]["sfp-temperature"]; ok {
				ft, err := strconv.ParseFloat(temp, 64)
				if err != nil {
					logger.Warn().Fields(map[string]any{"sfp-temperature": temp}).Err(err).Msg("extracting value from resource")
					continue
				}

				ir.sfpTemp.With(prometheus.Labels{"name": res[0]["name"]}).Set(ft)
			}
		}

	}

	return nil
}

// Collect implements Metric.
func (ir *InterfaceRate) StartCollecting(ctx context.Context) error {
	return startCollecting(ctx, ir)
}

// Register implements Metric.
func (ir *InterfaceRate) Register(ctx context.Context, constLabels prometheus.Labels, reg prometheus.Registerer) {
	ir.duplex = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   "mikrotik",
		Subsystem:   "interface",
		Name:        "full_duplex",
		Help:        "Full duplex data transmission",
		ConstLabels: constLabels,
	}, []string{"name"})
	reg.MustRegister(ir.duplex)

	ir.rate = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   "mikrotik",
		Subsystem:   "interface",
		Name:        "rate",
		Help:        "Actual interface connection data rate",
		ConstLabels: constLabels,
	}, []string{"name"})
	reg.MustRegister(ir.rate)

	ir.status = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   "mikrotik",
		Subsystem:   "interface",
		Name:        "status",
		Help:        "Current interface link status",
		ConstLabels: constLabels,
	}, []string{"name"})
	reg.MustRegister(ir.status)

	ir.sfpTemp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   "mikrotik",
		Subsystem:   "interface",
		Name:        "sfp_temperature",
		Help:        "Current SFP temperature",
		ConstLabels: constLabels,
	}, []string{"name"})
	reg.MustRegister(ir.sfpTemp)
}
