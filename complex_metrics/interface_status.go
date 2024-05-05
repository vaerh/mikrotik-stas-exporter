package complexmetrics

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
	"github.com/vaerh/mikrotik-prom-exporter/mikrotik"
)

func init() {
	ComplexMetrics.AddMetric(&InterfaceStatus{path: "/interface/ethernet", collectionInterval: DefaultMetricsCollectionInterval})
}

type InterfaceStatus struct {
	path               string
	duplex             *prometheus.GaugeVec
	rate               *prometheus.GaugeVec
	sfpTemp            *prometheus.GaugeVec
	status             *prometheus.GaugeVec
	collectionInterval time.Duration
}

func (iface *InterfaceStatus) GetCollectInterval() time.Duration {
	return iface.collectionInterval
}

func (iface *InterfaceStatus) SetCollectInterval(t time.Duration) {
	iface.collectionInterval = t
}

// Register implements Metric.
func (iface *InterfaceStatus) Register(ctx context.Context, constLabels prometheus.Labels, reg prometheus.Registerer) {
	iface.duplex = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   "mikrotik",
		Subsystem:   "interface",
		Name:        "full_duplex",
		Help:        "Full duplex data transmission",
		ConstLabels: constLabels,
	}, []string{"name"})
	reg.MustRegister(iface.duplex)

	iface.rate = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   "mikrotik",
		Subsystem:   "interface",
		Name:        "rate",
		Help:        "Actual interface connection data rate",
		ConstLabels: constLabels,
	}, []string{"name"})
	reg.MustRegister(iface.rate)

	iface.status = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   "mikrotik",
		Subsystem:   "interface",
		Name:        "status",
		Help:        "Current interface link status",
		ConstLabels: constLabels,
	}, []string{"name"})
	reg.MustRegister(iface.status)

	iface.sfpTemp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   "mikrotik",
		Subsystem:   "interface",
		Name:        "sfp_temperature",
		Help:        "Current SFP temperature",
		ConstLabels: constLabels,
	}, []string{"name"})
	reg.MustRegister(iface.sfpTemp)
}

// Collect implements Metric.
func (iface *InterfaceStatus) StartCollecting(ctx context.Context) error {
	return startCollecting(ctx, iface, iface.collect)
}

func (is *InterfaceStatus) collect(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Msg("exporting resources")

	// curl -s -k -X POST -H "content-type: application/json" "https://172.16.3.1/rest/interface/ethernet/monitor?once" --data '{"numbers":"ether1","once":""}'
	mikrotikResource, err := mikrotik.ReadResource(ctx, is.path, nil)
	if err != nil {
		return fmt.Errorf("reading resource: %w", err)
	}

	for _, iface := range mikrotikResource {
		var name string
		if c := iface["comment"]; c != "" {
			name = c
		} else {
			name = iface["name"]
		}

		// Get status
		if id, ok := iface[".id"]; ok {
			// Ethernet monitor
			res, err := mikrotik.Monitor(is.path, mikrotik.Ctx(ctx), map[string]string{
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
				is.status.With(prometheus.Labels{"name": name}).Set(1)
			} else {
				is.status.With(prometheus.Labels{"name": name}).Set(0)
			}

			switch res[0]["rate"] {
			case "10Mbps":
				is.rate.With(prometheus.Labels{"name": name}).Set(10)
			case "100Mbps":
				is.rate.With(prometheus.Labels{"name": name}).Set(100)
			case "1Gbps":
				is.rate.With(prometheus.Labels{"name": name}).Set(1000)
			case "2.5Gbps":
				is.rate.With(prometheus.Labels{"name": name}).Set(2500)
			case "5Gbps":
				is.rate.With(prometheus.Labels{"name": name}).Set(5000)
			case "10Gbps":
				is.rate.With(prometheus.Labels{"name": name}).Set(10000)
			case "40Gbps":
				is.rate.With(prometheus.Labels{"name": name}).Set(40000)
			default:
				is.rate.With(prometheus.Labels{"name": name}).Set(0)
			}

			if res[0]["full-duplex"] == "true" {
				is.duplex.With(prometheus.Labels{"name": name}).Set(1)
			} else {
				is.duplex.With(prometheus.Labels{"name": name}).Set(0)
			}

			if temp, ok := res[0]["sfp-temperature"]; ok {
				ft, err := strconv.ParseFloat(temp, 64)
				if err != nil {
					logger.Warn().Fields(map[string]any{"sfp-temperature": temp}).Err(err).Msg("extracting value from resource")
					continue
				}

				is.sfpTemp.With(prometheus.Labels{"name": name}).Set(ft)
			}
		}

	}

	return nil
}
