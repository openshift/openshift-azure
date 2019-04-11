// Package reporters implements gingo Reporter for  Azure App Insights
package reporters

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Microsoft/ApplicationInsights-Go/appinsights"
	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/types"
	"github.com/sirupsen/logrus"
)

type azureAppInsightsReporter struct {
	icli appinsights.TelemetryClient
	log  *logrus.Entry

	tracer *TraceWriter
}

var _ ginkgo.Reporter = &azureAppInsightsReporter{}

// NewAzureAppInsightsReporter returns reporter for Azure App Insights.
// If io.Reader is provided, it will send all the output from it to the
// AppInsights with test suite tag.
func NewAzureAppInsightsReporter(log *logrus.Entry, br io.Reader) ginkgo.Reporter {
	icli := appinsights.NewTelemetryClient(os.Getenv("AZURE_APP_INSIGHTS_KEY"))
	icli.Context().CommonProperties["type"] = "ginkgo"
	icli.Context().CommonProperties["resourcegroup"] = os.Getenv("RESOURCEGROUP")

	NewTraceWriter(icli, br)

	ar := &azureAppInsightsReporter{
		icli: icli,
		log:  log,
	}

	return ar
}

func (r *azureAppInsightsReporter) SpecSuiteWillBegin(config config.GinkgoConfigType, summary *types.SuiteSummary) {
}

func (r *azureAppInsightsReporter) BeforeSuiteDidRun(setupSummary *types.SetupSummary) {
}

func (r *azureAppInsightsReporter) AfterSuiteDidRun(setupSummary *types.SetupSummary) {
}

func (r *azureAppInsightsReporter) handleSetupSummary(name string, setupSummary *types.SetupSummary) {
}

func (r *azureAppInsightsReporter) SpecWillRun(specSummary *types.SpecSummary) {
}

func (r *azureAppInsightsReporter) SpecDidComplete(specSummary *types.SpecSummary) {
	result := map[string]interface{}{
		"ComponentTexts": strings.Join(specSummary.ComponentTexts, " "),
		"RunTime":        specSummary.RunTime.String(),
		"FailureMessage": specSummary.Failure.Message,
		"Failed":         specSummary.Failed(),
		"Passed":         specSummary.Passed(),
		"Skipped":        specSummary.Skipped(),
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		fmt.Println(err)
	}

	r.icli.TrackMetric(string(resultJSON), btof(specSummary.Failed()))
	// For debug comment out TrackEvent and output to stdout
	// fmt.Println(string(resultJSON))
}

func (r *azureAppInsightsReporter) SpecSuiteDidEnd(summary *types.SuiteSummary) {
}

func btof(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
