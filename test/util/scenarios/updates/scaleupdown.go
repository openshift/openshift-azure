//+build e2e

package updates

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/openshift/openshift-azure/test/util/client/azure"
	"github.com/openshift/openshift-azure/test/util/client/cluster"
	"github.com/openshift/openshift-azure/test/util/client/kubernetes"
)

func ResizeComputeScaleSet(az *azure.Client, manifest, configBlob string, size int) {
	By("Parsing the external manifest")
	external, err := cluster.ParseExternalConfig(manifest)
	Expect(err).NotTo(HaveOccurred())
	Expect(external).NotTo(BeNil())

	msg := fmt.Sprintf("Updating the size of the compute scale set to %d in the manifest", size)
	By(msg)
	Expect(external.Properties.AgentPoolProfiles[1].Name).To(Equal("compute"))
	external.Properties.AgentPoolProfiles[1].Count = size

	By("Calling update on the rp with the updated manifest")
	updated, err := az.UpdateCluster(external, configBlob, cluster.NewPluginConfig())
	Expect(err).NotTo(HaveOccurred())
	Expect(updated).NotTo(BeNil())
}

func CreateSampleOpenshiftDeployment(kc *kubernetes.Client, namespace, name string, replicas int) {
	err := kc.CreateSampleDeployment(namespace, name, replicas)
	Expect(err).NotTo(HaveOccurred())
}

func GetSampleOpenshiftDeployment(kc *kubernetes.Client, namespace, name string) (*v1.Deployment, error) {
	d, err := kc.GetDeployment(namespace, name)
	Expect(err).NotTo(HaveOccurred())
	Expect(d).NotTo(BeNil())
	return d, err
}

func GetSampleOpenshiftDeploymentPods(kc *kubernetes.Client, namespace, name string) ([]corev1.Pod, error) {
	pods, err := kc.GetDeploymentPods(namespace, name)
	Expect(err).NotTo(HaveOccurred())
	Expect(pods).NotTo(BeNil())
	Expect(len(pods)).To(BeNumerically(">=", 1))
	return pods, err
}

func DeleteSampleOpenshiftDeployment(kc *kubernetes.Client, namespace, name string) {
	err := kc.DeleteDeployment(namespace, name)
	Expect(err).NotTo(HaveOccurred())
}

func CheckDeploymentPodsDistribution(kc *kubernetes.Client, namespace, name string, podReplicas, usedNodes int) {
	dep, err := GetSampleOpenshiftDeployment(kc, namespace, name)
	Expect(err).NotTo(HaveOccurred())
	pods, err := GetSampleOpenshiftDeploymentPods(kc, dep.Namespace, dep.Name)
	Expect(err).NotTo(HaveOccurred())
	Expect(int(dep.Status.ReadyReplicas)).To(Equal(podReplicas))
	Expect(len(pods)).To(Equal(podReplicas))
	nodes := make(map[string]bool)
	for _, pod := range pods {
		nodes[pod.Spec.NodeName] = true
	}
	Expect(len(nodes)).To(Equal(usedNodes))
}
