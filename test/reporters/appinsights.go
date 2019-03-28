/*

Azure App Insights Reporter for Ginkgo
*/

package reporters

import (
	"fmt"
	"os"

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
	return &AzureAppInsightsReporter{
		icli: icli,
	}
}

func (r *AzureAppInsightsReporter) SpecSuiteWillBegin(config config.GinkgoConfigType, summary *types.SuiteSummary) {
}

func (r *AzureAppInsightsReporter) BeforeSuiteDidRun(setupSummary *types.SetupSummary) {
}

func (r *AzureAppInsightsReporter) AfterSuiteDidRun(setupSummary *types.SetupSummary) {
	r.icli.TrackEvent(fmt.Sprintf("%v | %v | %v ", setupSummary.SuiteID, setupSummary.RunTime, setupSummary.State == types.SpecStatePassed))

}

func (r *AzureAppInsightsReporter) handleSetupSummary(name string, setupSummary *types.SetupSummary) {
	if setupSummary.State != types.SpecStatePassed {
		// TODO: Track failures
	}
}

func (r *AzureAppInsightsReporter) SpecWillRun(specSummary *types.SpecSummary) {
}

func (r *AzureAppInsightsReporter) SpecDidComplete(specSummary *types.SpecSummary) {
}

func (r *AzureAppInsightsReporter) SpecSuiteDidEnd(summary *types.SuiteSummary) {
}
