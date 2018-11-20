//+build e2e

package customeradmin

import (
	"flag"

	. "github.com/onsi/ginkgo"

	"github.com/openshift/openshift-azure/test/util/client/kubernetes"
	"github.com/openshift/openshift-azure/test/util/scenarios/customercluster"
)

var (
	kc        *kubernetes.Client
	gitCommit = "unknown"

	kubeconfig  = flag.String("kubeconfig", "../../../_data/_out/customer-cluster-admin.kubeconfig", "Location of the kubeconfig")
	artifactDir = flag.String("artifact-dir", "../../../_data/_out/", "Directory to place artifacts when a test fails")
)

var _ = Describe("Openshift on Azure customer-cluster-admin e2e tests [CustomerAdmin]", func() {
	defer GinkgoRecover()

	It("should not read nodes", func() {
		customercluster.CheckCannotGetNode(kc)
	})

	It("should have full access on all non-infrastructure namespaces", func() {
		customercluster.CheckFullAccessToNonInfraNamespaces(kc)
	})

	It("should not list infra namespace secrets", func() {
		customercluster.CheckCannotListInfraSecrets(kc)
	})

	It("should not able to query groups", func() {
		customercluster.CheckCannotQueryCustomerAdminGroup(kc)
	})

	It("should not be able to escalate privileges", func() {
		customercluster.CheckCannotEscalatePrivilegeToClusterAdmin(kc)
	})

	// TODO: Test that a dedicated admin cannot delete pods in the default or openshift- namespaces
})
