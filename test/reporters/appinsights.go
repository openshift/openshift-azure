/*

Azure App Insights Reporter for Ginkgo
*/

package reporters

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/types"

	"github.com/openshift/openshift-azure/pkg/util/azureclient/insights"
)

type AzureAppInsightsReporter struct {
	icli insights.TelemetryClient
}

func NewAzureAppInsightsReporter() *AzureAppInsightsReporter {
	icli := insights.NewTelemetryClient(os.Getenv("AZURE_APP_INSIGHTS_KEY"))
	icli.Context().CommonProperties["type"] = "gotest"
	icli.Context().CommonProperties["resourcegroup"] = os.Getenv("RESOURCEGROUP")
	return &AzureAppInsightsReporter{
		icli: icli,
	}
}

func (r *AzureAppInsightsReporter) SpecSuiteWillBegin(config config.GinkgoConfigType, summary *types.SuiteSummary) {
}

func (r *AzureAppInsightsReporter) BeforeSuiteDidRun(setupSummary *types.SetupSummary) {
}

func (r *AzureAppInsightsReporter) AfterSuiteDidRun(setupSummary *types.SetupSummary) {
}

func (r *AzureAppInsightsReporter) handleSetupSummary(name string, setupSummary *types.SetupSummary) {
}

func (r *AzureAppInsightsReporter) SpecWillRun(specSummary *types.SpecSummary) {
}

func (r *AzureAppInsightsReporter) SpecDidComplete(specSummary *types.SpecSummary) {
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
	r.icli.TrackEvent(string(resultJSON))
	// For debug comment out TrackEvent and output to stdout
	// fmt.Println(string(resultJSON))
}

func (r *AzureAppInsightsReporter) SpecSuiteDidEnd(summary *types.SuiteSummary) {
}
