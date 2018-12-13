package specs

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sirupsen/logrus"
	apiappsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/util/randomstring"
	"github.com/openshift/openshift-azure/pkg/util/ready"
	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/clients/openshift"
)

var _ = Describe("Scale Up/Down E2E tests [ScaleUpDown][Fake][LongRunning]", func() {
	const (
		sampleDeployment = "hello-openshift"
	)
	var (
		azurecli          *azure.Client
		occli             *openshift.Client
		scaleUpManifest   = flag.String("scaleUpManifest", "../../test/manifests/normal/scaleup.yaml", "Path to the scale up manifest to send in a partial update request to the RP")
		scaleDownManifest = flag.String("scaleDownManifest", "../../test/manifests/normal/scaledown.yaml", "Path to the scale down manifest to send in a partial update request to the RP")
		namespace         string
	)

	BeforeEach(func() {
		var err error
		azurecli, err = azure.NewClientFromEnvironment()
		Expect(err).NotTo(HaveOccurred())
		occli, err = openshift.NewEndUserClient()
		Expect(err).NotTo(HaveOccurred())

		namespace, err = randomstring.RandomString("abcdefghijklmnopqrstuvwxyz0123456789", 5)
		Expect(err).ToNot(HaveOccurred())
		namespace = "e2e-test-" + namespace
		fmt.Fprintln(GinkgoWriter, "Using namespace", namespace)
		err = occli.CreateProject(namespace)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		occli.CleanupProject(namespace)
	})

	It("should be possible to maintain a healthy cluster after scaling it out and in", func() {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
		logrus.SetOutput(GinkgoWriter)
		log := logrus.NewEntry(logrus.StandardLogger())

		By("Fetching the scale up manifest")
		external, err := fakerp.LoadClusterConfigFromManifest(log, *scaleUpManifest)
		Expect(err).NotTo(HaveOccurred())
		Expect(external).NotTo(BeNil())

		By("Calling CreateOrUpdate on the rp with the scale up manifest")
		updated, err := azurecli.OpenShiftManagedClusters.CreateOrUpdateAndWait(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), *external)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).NotTo(BeNil())

		By("Creating a 2-replica sample deployment")
		int32Ptr := func(i int32) *int32 { return &i }
		deployment := &apiappsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sampleDeployment,
				Namespace: namespace,
			},
			Spec: apiappsv1.DeploymentSpec{
				Replicas: int32Ptr(int32(2)),
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
						NodeSelector: map[string]string{
							"node-role.kubernetes.io/compute": "true",
						},
						Containers: []corev1.Container{
							{
								Name:  sampleDeployment,
								Image: "openshift/" + sampleDeployment,
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										Protocol:      corev1.ProtocolTCP,
										ContainerPort: 8080,
									},
								},
							},
						},
					},
				},
			},
		}
		_, err = occli.AppsV1.Deployments(namespace).Create(deployment)
		Expect(err).NotTo(HaveOccurred())
		err = wait.PollImmediate(2*time.Second, 1*time.Minute, ready.DeploymentIsReady(occli.AppsV1.Deployments(namespace), sampleDeployment))
		Expect(err).NotTo(HaveOccurred())

		By("Verifying that the deployment's pods are spread across 2 nodes...")
		By(fmt.Sprintf("Getting deployment %s.%s", namespace, sampleDeployment))
		dep, err := occli.AppsV1.Deployments(namespace).Get(sampleDeployment, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(dep).NotTo(BeNil())

		By(fmt.Sprintf("Getting %s.%s deployment's labels", namespace, sampleDeployment))
		set := labels.Set(dep.Spec.Template.Labels)
		listOptions := metav1.ListOptions{LabelSelector: set.AsSelector().String()}

		By(fmt.Sprintf("Listing pods matching deployment's labels %+v", set))
		poditems, err := occli.CoreV1.Pods(namespace).List(listOptions)
		Expect(err).NotTo(HaveOccurred())
		Expect(poditems).NotTo(BeNil())
		pods := poditems.Items
		Expect(int(dep.Status.ReadyReplicas)).To(Equal(2))
		Expect(len(pods)).To(Equal(2))
		nodes := make(map[string]bool)
		for _, pod := range pods {
			By(fmt.Sprintf("Found pod %s on node %s", pod.Name, pod.Spec.NodeName))
			nodes[pod.Spec.NodeName] = true
		}
		Expect(len(nodes)).To(Equal(2))

		By("Fetching the scale down manifest")
		external, err = fakerp.LoadClusterConfigFromManifest(log, *scaleDownManifest)
		Expect(err).NotTo(HaveOccurred())
		Expect(external).NotTo(BeNil())

		By("Calling CreateOrUpdate on the rp with the scale down manifest")
		updated, err = azurecli.OpenShiftManagedClusters.CreateOrUpdateAndWait(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), *external)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).NotTo(BeNil())

		By("Verifying that the deployment's pods are all on 1 node...")
		By(fmt.Sprintf("Listing pods matching deployment's labels %+v", set))
		poditems, err = occli.CoreV1.Pods(namespace).List(listOptions)
		Expect(err).NotTo(HaveOccurred())
		Expect(poditems).NotTo(BeNil())
		pods = poditems.Items
		Expect(len(pods)).To(Equal(2))
		nodes = make(map[string]bool)
		for _, pod := range pods {
			Expect(pod.Spec.NodeName).NotTo(BeEmpty())
			By(fmt.Sprintf("Found pod %s on node %s", pod.Name, pod.Spec.NodeName))
			nodes[pod.Spec.NodeName] = true
			Expect(pod.Status.Phase).To(Equal(corev1.PodRunning))
		}
		Expect(len(nodes)).To(Equal(1))
	})
})
