//+build e2e

package azurereader

import (
	"flag"

	. "github.com/onsi/ginkgo"

	"github.com/openshift/openshift-azure/test/util/client/kubernetes"
	"github.com/openshift/openshift-azure/test/util/scenarios/clusterreader"
)

var (
	kc        *kubernetes.Client
	gitCommit = "unknown"

	artifactDir = flag.String("artifact-dir", "../../../_data/_out/", "Directory to place artifacts when a test fails")
)

var _ = Describe("Openshift on Azure admin e2e tests [AzureClusterReader]", func() {
	defer GinkgoRecover()

	It("should label nodes correctly", func() {
		clusterreader.CheckNodesLabelledCorrectly(kc)
	})

	It("should start prometheus correctly", func() {
		clusterreader.CheckPrometheusStartedCorrectly(kc)
	})

	It("should run the correct image", func() {
		clusterreader.CheckCorrectImageWasUsed(kc)
	})
})
