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
	ComplexMetrics.AddMetric(&PoEStatus{path: "/interface/ethernet/poe"})
}

type PoEStatus struct {
	path          string
	status        *prometheus.GaugeVec
	outputCurrent *prometheus.GaugeVec
	outputPower   *prometheus.GaugeVec
	outputVoltage *prometheus.GaugeVec
}

// Register implements Metric.
func (poe *PoEStatus) Register(ctx context.Context, constLabels prometheus.Labels, reg prometheus.Registerer) {
	poe.status = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   "mikrotik",
		Subsystem:   "interface",
		Name:        "ethernet_poe_status",
		Help:        "PoE status",
		ConstLabels: constLabels,
	}, []string{"name", "poe_out", "poe_priority"})
	reg.MustRegister(poe.status)

	poe.outputCurrent = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   "mikrotik",
		Subsystem:   "interface",
		Name:        "ethernet_poe_current",
		Help:        "Current (mA)",
		ConstLabels: constLabels,
	}, []string{"name"})
	reg.MustRegister(poe.outputCurrent)

	poe.outputPower = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   "mikrotik",
		Subsystem:   "interface",
		Name:        "ethernet_poe_power",
		Help:        "Power (W)",
		ConstLabels: constLabels,
	}, []string{"name"})
	reg.MustRegister(poe.outputPower)

	poe.outputVoltage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace:   "mikrotik",
		Subsystem:   "interface",
		Name:        "ethernet_poe_voltage",
		Help:        "Voltage (V)",
		ConstLabels: constLabels,
	}, []string{"name"})
	reg.MustRegister(poe.outputVoltage)
}

// Collect implements Metric.
func (poe *PoEStatus) StartCollecting(ctx context.Context) error {
	return startCollecting(ctx, poe.collect)
}

func (poe *PoEStatus) collect(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Msg("exporting resources")

	mikrotikResource, err := mikrotik.ReadResource(ctx, poe.path, nil)
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
			res, err := mikrotik.Monitor(poe.path, mikrotik.Ctx(ctx), map[string]string{
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

			// PoE monitor
			res, err = mikrotik.Monitor(poe.path, mikrotik.Ctx(ctx), map[string]string{
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

			if res[0]["poe-out-status"] == "powered-on" {
				poe.status.With(prometheus.Labels{"name": name, "poe_out": res[0]["poe-out"], "poe_priority": iface["poe-priority"]}).Set(1)
			} else {
				poe.status.With(prometheus.Labels{"name": name, "poe_out": res[0]["poe-out"], "poe_priority": iface["poe-priority"]}).Set(0)
			}

			if curr, ok := res[0]["poe-out-current"]; ok {
				f, err := strconv.ParseFloat(curr, 64)
				if err != nil {
					logger.Warn().Fields(map[string]any{"poe-out-current": curr}).Err(err).Msg("extracting value from resource")
					continue
				}

				poe.outputCurrent.With(prometheus.Labels{"name": name}).Set(f)
			} else {
				poe.outputCurrent.With(prometheus.Labels{"name": name}).Set(0)
			}

			if curr, ok := res[0]["poe-out-power"]; ok {
				f, err := strconv.ParseFloat(curr, 64)
				if err != nil {
					logger.Warn().Fields(map[string]any{"poe-out-power": curr}).Err(err).Msg("extracting value from resource")
					continue
				}

				poe.outputPower.With(prometheus.Labels{"name": name}).Set(f)
			} else {
				poe.outputPower.With(prometheus.Labels{"name": name}).Set(0)
			}

			if curr, ok := res[0]["poe-out-voltage"]; ok {
				f, err := strconv.ParseFloat(curr, 64)
				if err != nil {
					logger.Warn().Fields(map[string]any{"poe-out-voltage": curr}).Err(err).Msg("extracting value from resource")
					continue
				}

				poe.outputVoltage.With(prometheus.Labels{"name": name}).Set(f)
			} else {
				poe.outputVoltage.With(prometheus.Labels{"name": name}).Set(0)
			}
		}

	}

	return nil
}
