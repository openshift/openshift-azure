package customeradmin

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	infoGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_controllers_info",
			Help: "General information about the azure controllers.",
		},
		[]string{"name", "image", "period_seconds"},
	)

	errorsCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "azure_controllers_errors_total",
			Help: "Total number of errors.",
		},
		[]string{"name"},
	)

	inFlightGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_controllers_reconciliations_inflight",
			Help: "Number of azure controller reconciliations in progress.",
		},
		[]string{"name"},
	)

	lastExecutedGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_controllers_last_executed",
			Help: "The last time the azure controllers were run.",
		},
		[]string{"name"},
	)

	durationSummary = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "azure_controllers_duration_seconds",
			Help: "The duration of azure controller runs.",
		},
		[]string{"name"},
	)
)
