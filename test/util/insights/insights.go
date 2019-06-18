// Package reporters implements gingo Reporter for  Azure App Insights
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
	action   string
	duration time.Duration
	result   bool

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
	return nil, fmt.Errorf("AZURE_APP_INSIGHTS_KEY variable not set")
}

func (r *azureAppInsightsReporter) Start(action string) {
	r.metrics.start = time.Now()
	r.metrics.action = action
}

func (r *azureAppInsightsReporter) Stop(result bool) {
	r.metrics.duration = time.Since(r.metrics.start)
	data := map[string]interface{}{
		"action":   r.metrics.action,
		"duration": r.metrics.duration,
	}
	dataJSON, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
	}

	r.c.TrackMetric(string(dataJSON), btof(r.metrics.result))
	// For debug comment out TrackEvent and output to stdout
	// fmt.Println(string(resultJSON))
}

func btof(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
