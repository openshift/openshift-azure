package insights

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
)

type azureAppInsightsReporter struct {
	c       appinsights.TelemetryClient
	metrics metrics
}

type metrics struct {
	action string
	result bool

	start time.Time
}

// NewAzureAppInsightsReporter returns reporter for Azure App Insights.
func NewAzureAppInsightsReporter() (*azureAppInsightsReporter, error) {
	if os.Getenv("AZURE_APP_INSIGHTS_KEY") != "" {
		c := appinsights.NewTelemetryClient(os.Getenv("AZURE_APP_INSIGHTS_KEY"))
		c.Context().CommonProperties["type"] = "install"
		c.Context().CommonProperties["version"] = "v4"
		c.Context().CommonProperties["resourcegroup"] = os.Getenv("RESOURCEGROUP")
		return &azureAppInsightsReporter{
			c:       c,
			metrics: metrics{},
		}, nil
	}
	return nil, fmt.Errorf("AZURE_APP_INSIGHTS_KEY not set")
}

func (r *azureAppInsightsReporter) Start(action string) {
	r.metrics.start = time.Now()
	r.metrics.action = action
}

func (r *azureAppInsightsReporter) Stop(result bool) error {
	data := map[string]interface{}{
		"action":   r.metrics.action,
		"duration": time.Since(r.metrics.start),
	}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}
	r.c.TrackMetric(string(dataJSON), btof(result))
	// For debug comment out TrackEvent and output to stdout
	fmt.Println(string(dataJSON))

	return nil
}

func btof(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
