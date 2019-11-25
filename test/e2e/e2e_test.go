//+build e2e

package e2e

import (
	"flag"
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"github.com/onsi/gomega/format"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/fakerp"
	_ "github.com/openshift/openshift-azure/test/e2e/specs"
	_ "github.com/openshift/openshift-azure/test/e2e/specs/fakerp"
	"github.com/openshift/openshift-azure/test/reporters"
)

var (
	gitCommit = "unknown"
)

func TestE2E(t *testing.T) {
	fmt.Printf("e2e tests starting, git commit %s\n", gitCommit)

	flag.Parse()
	logrus.SetFormatter(&logrus.TextFormatter{})
	logrus.SetOutput(GinkgoWriter)
	logrus.SetReportCaller(true)

	err := fakerp.ConfigureProxyDialer()
	if err != nil {
		t.Fatal(err)
	}

	RegisterFailHandler(Fail)

	c := appinsights.NewTelemetryClient(os.Getenv("AZURE_APP_INSIGHTS_KEY"))
	c.Context().CommonProperties["type"] = "ginkgo"
	c.Context().CommonProperties["resourcegroup"] = os.Getenv("RESOURCEGROUP")

	format.TruncatedDiff = false

	if os.Getenv("AZURE_APP_INSIGHTS_KEY") != "" {
		ar := reporters.NewAzureAppInsightsReporter(c)

		doneStdout := make(chan struct{})
		captureStdout, err := reporters.StartCapture(1, c, doneStdout)
		if err != nil {
			t.Fatal(err)
		}
		doneStderr := make(chan struct{})
		captureStderr, err := reporters.StartCapture(2, c, doneStderr)
		if err != nil {
			t.Fatal(err)
		}

		RunSpecsWithDefaultAndCustomReporters(t, "e2e tests", []Reporter{ar})
		captureStdout.Close()
		captureStderr.Close()
		<-doneStdout
		<-doneStderr
	} else {
		RunSpecs(t, "e2e tests")
	}
}
