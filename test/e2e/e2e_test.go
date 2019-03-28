//+build e2e

package e2e

import (
	"flag"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sirupsen/logrus"

	_ "github.com/openshift/openshift-azure/test/e2e/specs"
	_ "github.com/openshift/openshift-azure/test/e2e/specs/fakerp"
	_ "github.com/openshift/openshift-azure/test/e2e/specs/realrp"
	azreporters "github.com/openshift/openshift-azure/test/reporters"
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

	RegisterFailHandler(Fail)

	var reporters []Reporter
	azureReporters := append(reporters, azreporters.NewAzureAppInsightsReporter())
	RunSpecsWithDefaultAndCustomReporters(t, "e2e tests", azureReporters)
}
