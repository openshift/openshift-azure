package insights

import (
	"time"

	"github.com/Microsoft/ApplicationInsights-Go/appinsights"
)

// TelemetryClient is a minimal interface for azure TelemetryClient
type TelemetryClient interface {
	Track(appinsights.Telemetry)
	TrackEvent(name string)
	TrackMetric(name string, value float64)
	TrackAvailability(name string, duration time.Duration, success bool)
}

type telemetryClient struct {
	appinsights.TelemetryClient
}

var _ TelemetryClient = &telemetryClient{}

func NewTelemetryClient(insightsKey string) appinsights.TelemetryClient {
	return appinsights.NewTelemetryClient(insightsKey)
}

func NewRequestTelemetry(method, uri string, duration time.Duration, responseCode string) *appinsights.RequestTelemetry {
	return appinsights.NewRequestTelemetry(method, uri, duration, responseCode)
}
