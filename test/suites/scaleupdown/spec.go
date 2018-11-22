//+build e2e

package scaleupdown

import (
	"flag"

	. "github.com/onsi/ginkgo"

	"github.com/openshift/openshift-azure/test/util/client/azure"
	"github.com/openshift/openshift-azure/test/util/client/kubernetes"
	"github.com/openshift/openshift-azure/test/util/scenarios/updates"
)

var (
	kc        *kubernetes.Client
	az        *azure.Client
	gitCommit = "unknown"

	manifest    = flag.String("manifest", "../../../_data/manifest.yaml", "Path to the manifest to send to the RP")
	configBlob  = flag.String("configBlob", "../../../_data/containerservice.yaml", "Path on disk where the OpenShift internal config blob should be written")
	kubeconfig  = flag.String("kubeconfig", "../../../_data/_out/admin.kubeconfig", "Location of the kubeconfig")
	artifactDir = flag.String("artifact-dir", "../../../_data/_out/", "Directory to place artifacts when a test fails")
)

var _ = Describe("Scale Out/In E2E tests [Fake] [Real] [Update]", func() {
	defer GinkgoRecover()

	namespace := "default"
	deploymentName := "hello-e2e"
	podReplicas := 2

	It("should scale out the compute scale set to 2 vms", func() {
		updates.ResizeComputeScaleSet(az, *manifest, *configBlob, podReplicas)
	})

	It("should create a 2-replica sample deployment", func() {
		updates.CreateSampleOpenshiftDeployment(kc, namespace, deploymentName, podReplicas)
	})

	It("should get the deployment's pods and check that they are spread across nodes", func() {
		updates.CheckDeploymentPodsDistribution(kc, namespace, deploymentName, podReplicas, podReplicas)
	})

	It("should scale in the compute scale set to 1 vm", func() {
		updates.ResizeComputeScaleSet(az, *manifest, *configBlob, 1)
	})

	It("should get the deployment's pods and check that they are back on a single node", func() {
		updates.CheckDeploymentPodsDistribution(kc, namespace, deploymentName, podReplicas, 1)
	})

	It("should delete the sample deployment", func() {
		updates.DeleteSampleOpenshiftDeployment(kc, namespace, deploymentName)
	})
})
