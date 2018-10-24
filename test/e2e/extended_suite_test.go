//+build e2e

package e2e

import (
	"flag"
	"fmt"
	"testing"

	"github.com/kelseyhightower/envconfig"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
)

var (
	gitCommit   = "unknown"
	testConfig  Config
	artifactDir = flag.String("artifact-dir", "", "Directory to place artifacts when a test fails")
)

var _ = BeforeSuite(func() {
	err := envconfig.Process("", &testConfig)
	if err != nil {
		panic(err)
	}
	testConfig.ArtifactDir = *artifactDir

	suiteSummary := config.GinkgoConfig.FocusString
	if inFocus(suiteSummary, "CustomerAdmin") {
		c = newTestClient(testConfig.KubeConfig, "enduser", testConfig.ArtifactDir)
		creader = newTestClient(testConfig.KubeConfig, "customer-cluster-reader", testConfig.ArtifactDir)
		cadmin = newTestClient(testConfig.KubeConfig, "customer-cluster-admin", testConfig.ArtifactDir)
		return
	}
	if inFocus(suiteSummary, "AzureClusterReader") {
		c = newTestClient(testConfig.KubeConfig, "", testConfig.ArtifactDir)
		return
	}
	c = newTestClient(testConfig.KubeConfig, "", testConfig.ArtifactDir)
})

func TestExtended(t *testing.T) {
	flag.Parse()
	fmt.Printf("e2e tests starting, git commit %s\n", gitCommit)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Extended Suite")
}
