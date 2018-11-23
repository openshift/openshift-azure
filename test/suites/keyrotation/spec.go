//+build e2e

package keyrotation

import (
	"flag"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/test/util/client/azure"
	"github.com/openshift/openshift-azure/test/util/client/kubernetes"
	"github.com/openshift/openshift-azure/test/util/scenarios/enduser"
	"github.com/openshift/openshift-azure/test/util/scenarios/updates"
)

var (
	kc        *kubernetes.Client
	az        *azure.Client
	gitCommit = "unknown"

	manifest    = flag.String("manifest", "../../../_data/manifest.yaml", "Path to the manifest to send to the RP")
	configBlob  = flag.String("configBlob", "../../../_data/containerservice.yaml", "Path on disk where the OpenShift internal config blob should be written")
	artifactDir = flag.String("artifact-dir", "../../../_data/_out/", "Directory to place artifacts when a test fails")
)

var _ = Describe("Key Rotation E2E tests [Fake] [Update]", func() {
	defer GinkgoRecover()

	It("should be possible to maintain a healthy cluster after rotating all credentials", func() {
		updates.RotateClusterCredentials(az, *manifest, *configBlob)
	})

	Context("when the key rotation is successful", func() {
		BeforeEach(func() {
			namespace := kc.GenerateRandomName("e2e-test-")
			kc.CreateProject(namespace)
		})

		AfterEach(func() {
			if CurrentGinkgoTestDescription().Failed {
				if err := kc.DumpInfo(); err != nil {
					logrus.Warn(err)
				}
			}
			kc.CleanupProject(10 * time.Minute)
		})

		It("should disallow PDB mutations", func() {
			enduser.CheckPdbMutationsDisallowed(kc)
		})

		It("should deploy a template and ensure a given text is in the contents", func() {
			enduser.CheckCanDeployTemplate(kc)
		})

		It("should not be possible to crud infra resources", func() {
			enduser.CheckCrudOnInfraDisallowed(kc)
		})

		It("should deploy a template with persistent storage and test failure modes", func() {
			enduser.CheckCanDeployTemplateWithPV(kc)
		})
	})
})
