package metricsbridge

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricsBridgeRegistry = prometheus.NewRegistry()

	// TODO(charlesakalugwu): Add unit tests for the handling of these metrics once
	//  the upstream library supports it
	metricsBridgeErrorsCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "metrics_bridge_errors_total",
			Help: "Total number of errors.",
		},
	)

	metricsBridgeBytesTransferredCounter = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "metrics_bridge_bytes_transferred_total",
			Help: "Total number of bytes transferred.",
		},
	)

	metricsBridgeMetricsTransferredCounter = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "metrics_bridge_metrics_transferred_total",
			Help: "Total number of metrics transferred.",
		},
	)

	metricsBridgeProcessingDurationSummary = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Name: "metrics_bridge_processing_duration_seconds",
			Help: "The duration of metrics bridge query processing.",
		},
	)
)

func init() {
	metricsBridgeRegistry.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)
	metricsBridgeRegistry.MustRegister(metricsBridgeErrorsCounter)
	metricsBridgeRegistry.MustRegister(metricsBridgeBytesTransferredCounter)
	metricsBridgeRegistry.MustRegister(metricsBridgeMetricsTransferredCounter)
	metricsBridgeRegistry.MustRegister(metricsBridgeProcessingDurationSummary)
}

func MetricsHandler() http.Handler {
	return promhttp.HandlerFor(metricsBridgeRegistry, promhttp.HandlerOpts{})
}
