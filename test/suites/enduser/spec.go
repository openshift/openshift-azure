//+build e2e

package enduser

import (
	"flag"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/test/util/client/kubernetes"
	"github.com/openshift/openshift-azure/test/util/scenarios/enduser"
)

var (
	c           *kubernetes.Client
	gitCommit   = "unknown"
	kubeconfig  = flag.String("kubeconfig", "../../../_data/_out/enduser.kubeconfig", "Location of the kubeconfig")
	artifactDir = flag.String("artifact-dir", "../../../_data/_out/", "Directory to place artifacts when a test fails")
)

var _ = Describe("Openshift on Azure end user e2e tests [EndUser]", func() {
	defer GinkgoRecover()

	BeforeEach(func() {
		namespace := c.GenerateRandomName("e2e-test-")
		c.CreateProject(namespace)
	})

	AfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			if err := c.DumpInfo(); err != nil {
				logrus.Warn(err)
			}
		}
		c.CleanupProject(10 * time.Minute)
	})

	It("should disallow PDB mutations", func() {
		enduser.CheckPdbMutationsDisallowed(c)
	})

	It("should deploy a template and ensure a given text is in the contents", func() {
		enduser.CheckCanDeployTemplate(c)
	})

	It("should not crud infra resources", func() {
		enduser.CheckCrudOnInfraDisallowed(c)
	})

	It("should deploy a template with persistent storage and test failure modes", func() {
		enduser.CheckCanDeployTemplateWithPV(c)
	})
})
