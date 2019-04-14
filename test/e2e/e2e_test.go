package e2e

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/log"
	_ "github.com/openshift/openshift-azure/test/e2e/specs"
	_ "github.com/openshift/openshift-azure/test/e2e/specs/fakerp"
	_ "github.com/openshift/openshift-azure/test/e2e/specs/realrp"
	"github.com/openshift/openshift-azure/test/reporters"
	"github.com/openshift/openshift-azure/test/sanity"
)

var (
	gitCommit = "unknown"
)

func TestE2E(t *testing.T) {
	logger := os.Stdout
	sanity.GlobalLogger = logger

	fd := int(logger.Fd())
	capture, err := reporters.NewCapture(fd)
	if err != nil {
		panic(err)
	}

	logrus.SetLevel(log.SanitizeLogLevel("Debug"))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logrus.SetOutput(sanity.GlobalLogger)
	log := logrus.NewEntry(logrus.StandardLogger())

	RegisterFailHandler(Fail)

	// init reporter to see logs early
	ar := reporters.NewAzureAppInsightsReporter(log, capture.Reader)

	log.Debugf("e2e tests starting, git commit %s", gitCommit)

	// IMPORTANT: Current AzureAppInsight reported does not support parallel tests
	// This is due inability to distinguish betwean tests cases in the output.
	// If at any point parallel execution is needed, GinkoWriter should be initiated
	// at the spec level, and potentially handled via multiple buffers
	if os.Getenv("AZURE_APP_INSIGHTS_KEY") != "" {
		RunSpecsWithDefaultAndCustomReporters(t, "e2e tests", []Reporter{ar})
	} else {
		RunSpecs(t, "e2e tests")
	}
}
