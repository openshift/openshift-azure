package customeradmin

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	azureControllersRegistry = prometheus.NewRegistry()

	// TODO(charlesakalugwu): Add unit tests for the handling of these metrics once
	//  the upstream library supports it
	azureControllersErrorsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "azure_controllers_errors_total",
			Help: "Total number of errors.",
		},
		[]string{"controller"},
	)

	azureControllersInFlightGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_controllers_reconciliations_inflight",
			Help: "Number of azure controller reconciliations in progress.",
		},
		[]string{"controller"},
	)

	azureControllersLastExecutedGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_controllers_last_executed",
			Help: "The last time the azure controllers were run.",
		},
		[]string{"controller"},
	)

	azureControllersDurationSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "azure_controllers_duration_seconds",
			Help: "The duration of azure controller runs.",
		},
		[]string{"controller"},
	)
)

func init() {
	azureControllersRegistry.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)
	azureControllersRegistry.MustRegister(azureControllersErrorsCounter)
	azureControllersRegistry.MustRegister(azureControllersInFlightGauge)
	azureControllersRegistry.MustRegister(azureControllersLastExecutedGauge)
	azureControllersRegistry.MustRegister(azureControllersDurationSummary)

	// initialize metrics with known label values to 0 otherwise their time series
	// will be missing until they are used
	for _, controller := range knownControllers {
		azureControllersErrorsCounter.WithLabelValues(controller)
		azureControllersInFlightGauge.WithLabelValues(controller)
		azureControllersLastExecutedGauge.WithLabelValues(controller)
		azureControllersDurationSummary.WithLabelValues(controller)
	}
}

func MetricsHandler() http.Handler {
	return promhttp.HandlerFor(azureControllersRegistry, promhttp.HandlerOpts{})
}
