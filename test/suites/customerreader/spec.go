//+build e2e

package customerreader

import (
	"flag"

	. "github.com/onsi/ginkgo"

	"github.com/openshift/openshift-azure/test/util/client/kubernetes"
	"github.com/openshift/openshift-azure/test/util/scenarios/customercluster"
)

var (
	kc        *kubernetes.Client
	gitCommit = "unknown"

	artifactDir = flag.String("artifact-dir", "../../../_data/_out/", "Directory to place artifacts when a test fails")
)

var _ = Describe("Openshift on Azure customer-cluster-reader e2e tests [CustomerReader]", func() {
	defer GinkgoRecover()

	It("should not read nodes", func() {
		customercluster.CheckCannotGetNode(kc)
	})

	It("should have read-only on all non-infrastructure namespaces", func() {
		customercluster.CheckReadOnlyAccessToNonInfraNamespaces(kc)
	})

	It("should not list infra namespace secrets", func() {
		customercluster.CheckCannotListInfraSecrets(kc)
	})

	It("should not be able to query groups", func() {
		customercluster.CheckCannotQueryCustomerReaderGroup(kc)
	})
})
