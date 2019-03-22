package specs

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/go-autorest/autorest/to"
	apiappsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/util/random"
	"github.com/openshift/openshift-azure/pkg/util/ready"
	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/sanity"
)

var _ = Describe("Scale Up/Down E2E tests [ScaleUpDown][Fake][LongRunning]", func() {
	const (
		sampleDeployment = "hello-openshift"
	)
	var (
		azurecli  *azure.Client
		namespace string
	)

	BeforeEach(func() {
		var err error
		azurecli, err = azure.NewClientFromEnvironment(true)
		Expect(err).NotTo(HaveOccurred())
		Expect(azurecli).NotTo(BeNil())

		namespace, err = random.LowerCaseAlphanumericString(5)
		Expect(err).ToNot(HaveOccurred())
		namespace = "e2e-test-" + namespace
		fmt.Fprintln(GinkgoWriter, "Using namespace", namespace)
		err = sanity.Checker.Client.EndUser.CreateProject(namespace)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		sanity.Checker.Client.EndUser.CleanupProject(namespace)
	})

	scale := func(ubs updateblob.BlobService, before *updateblob.UpdateBlob, count int64) {
		By("Fetching the manifest")
		external, err := azurecli.OpenShiftManagedClusters.Get(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		external.Properties.ProvisioningState = nil // TODO: should not need to do this
		err = setCount(&external, count)
		Expect(err).NotTo(HaveOccurred())

		By("Calling CreateOrUpdate on the rp with the scale up manifest")
		_, err = azurecli.OpenShiftManagedClusters.CreateOrUpdateAndWait(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), external)
		Expect(err).NotTo(HaveOccurred())

		By("Reading the update blob after the scale up")
		after, err := ubs.Read()
		Expect(err).ToNot(HaveOccurred())

		By("Verifying that the update blob is unchanged")
		Expect(before).To(Equal(after))

		By("Verifying that the expected number of nodes exist and are ready")
		nodes, err := sanity.Checker.Client.Admin.CoreV1.Nodes().List(metav1.ListOptions{})
		var nodeCount int64
		for _, node := range nodes.Items {
			if !strings.HasPrefix(node.Name, "master-") &&
				!strings.HasPrefix(node.Name, "infra-") &&
				ready.NodeIsReady(&node) {
				nodeCount++
			}
		}
		Expect(nodeCount).To(Equal(count))

		By("Validating the cluster")
		errs := sanity.Checker.ValidateCluster(context.Background())
		Expect(errs).To(BeEmpty())
	}

	It("should be possible to maintain a healthy cluster after scaling it out and in", func() {
		By("Reading the update blob before the scales")
		ubs := updateblob.NewBlobService(azurecli.BlobStorage)
		before, err := ubs.Read()
		Expect(err).ToNot(HaveOccurred())
		Expect(len(before.HostnameHashes)).To(Equal(3)) // one per master instance
		Expect(len(before.ScalesetHashes)).To(Equal(2)) // one per worker scaleset

		By("Scaling up")
		scale(ubs, before, 2)

		By("Creating a 2-replica sample deployment")
		d := &apiappsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sampleDeployment,
				Namespace: namespace,
			},
			Spec: apiappsv1.DeploymentSpec{
				Replicas: to.Int32Ptr(2),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": sampleDeployment,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name: sampleDeployment,
						Labels: map[string]string{
							"app": sampleDeployment,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  sampleDeployment,
								Image: "openshift/" + sampleDeployment,
							},
						},
					},
				},
			},
		}
		_, err = sanity.Checker.Client.EndUser.AppsV1.Deployments(namespace).Create(d)
		Expect(err).NotTo(HaveOccurred())

		By("Verifying that the deployment's pods are spread across 2 nodes")
		err = wait.PollImmediate(2*time.Second, 1*time.Minute, ready.CheckDeploymentIsReady(sanity.Checker.Client.EndUser.AppsV1.Deployments(namespace), sampleDeployment))
		Expect(err).NotTo(HaveOccurred())
		d, err = sanity.Checker.Client.EndUser.AppsV1.Deployments(namespace).Get(sampleDeployment, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		pods, err := sanity.Checker.Client.EndUser.CoreV1.Pods(namespace).List(metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(d.Status.ReadyReplicas).To(BeEquivalentTo(2))
		Expect(len(pods.Items)).To(Equal(2))
		Expect(pods.Items[0].Spec.NodeName).NotTo(Equal(pods.Items[1].Spec.NodeName))

		By("Scaling down")
		scale(ubs, before, 1)

		By("Verifying that the deployment's pods are all on 1 node")
		err = wait.PollImmediate(2*time.Second, 1*time.Minute, ready.CheckDeploymentIsReady(sanity.Checker.Client.EndUser.AppsV1.Deployments(namespace), sampleDeployment))
		Expect(err).NotTo(HaveOccurred())
		d, err = sanity.Checker.Client.EndUser.AppsV1.Deployments(namespace).Get(sampleDeployment, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		pods, err = sanity.Checker.Client.EndUser.CoreV1.Pods(namespace).List(metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(d.Status.ReadyReplicas).To(BeEquivalentTo(2))
		Expect(len(pods.Items)).To(Equal(2))
		Expect(pods.Items[0].Spec.NodeName).To(Equal(pods.Items[1].Spec.NodeName))
	})
})

func setCount(oc *v20180930preview.OpenShiftManagedCluster, count int64) error {
	for _, p := range oc.Properties.AgentPoolProfiles {
		if *p.Role == v20180930preview.AgentPoolProfileRoleCompute {
			*p.Count = count
			return nil
		}
	}
	return fmt.Errorf("compute agent pool profile not found")
}
