package fakerp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/ghodss/yaml"
	templatev1 "github.com/openshift/api/template/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/util/jsonpath"
	"github.com/openshift/openshift-azure/pkg/util/random"
	"github.com/openshift/openshift-azure/pkg/util/ready"
	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/sanity"
	"github.com/openshift/openshift-azure/test/util/exec"
)

var _ = Describe("Openshift on Azure admin e2e tests [EveryPR]", func() {
	It("should run the correct image", func() {
		// e2e check should ensure that no reg-aws images are running on box
		pods, err := sanity.Checker.Client.Admin.CoreV1.Pods("").List(metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())

		for _, pod := range pods.Items {
			for _, container := range pod.Spec.Containers {
				Expect(strings.HasPrefix(container.Image, "registry.reg-aws.openshift.com/")).ToNot(BeTrue())
			}
		}

		// fetch master-000000 and determine the OS type
		master0, _ := sanity.Checker.Client.Admin.CoreV1.Nodes().Get("master-000000", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		// set registryPrefix to appropriate string based upon master's OS type
		var registryPrefix string
		if strings.HasPrefix(master0.Status.NodeInfo.OSImage, "Red Hat Enterprise") {
			registryPrefix = "registry.access.redhat.com/openshift3/ose-"
		} else {
			registryPrefix = "quay.io/openshift/origin-"
		}

		// Check all Configmaps for image format matches master's OS type
		// format: registry.access.redhat.com/openshift3/ose-${component}:${version}
		configmaps, err := sanity.Checker.Client.Admin.CoreV1.ConfigMaps("openshift-node").List(metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		var nodeConfig map[string]interface{}
		for _, configmap := range configmaps.Items {
			err = yaml.Unmarshal([]byte(configmap.Data["node-config.yaml"]), &nodeConfig)
			format := jsonpath.MustCompile("$.imageConfig.format").MustGetString(nodeConfig)
			Expect(strings.HasPrefix(format, registryPrefix)).To(BeTrue())
		}
	})

	It("Should check that pods cannot access the Kubelets' read-only port on its VM's default network interface", func() {
		nginxTemplate := "nginx-example"

		namespace, err := random.LowerCaseAlphanumericString(5)
		Expect(err).ToNot(HaveOccurred())
		namespace = "e2e-test-" + namespace

		By(fmt.Sprintf("creating namespace %s", namespace))
		err = sanity.Checker.Client.EndUser.CreateProject(namespace)
		Expect(err).ToNot(HaveOccurred())
		//defer sanity.Checker.Client.EndUser.CleanupProject(namespace)

		By(fmt.Sprintf("getting the %s template", nginxTemplate))
		template, err := sanity.Checker.Client.Admin.TemplateV1.Templates("openshift").Get(nginxTemplate, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		By(fmt.Sprintf("deploying %s in the %s namespace", nginxTemplate, namespace))
		_, err = sanity.Checker.Client.Admin.TemplateV1.TemplateInstances(namespace).Create(
			&templatev1.TemplateInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
				Spec: templatev1.TemplateInstanceSpec{
					Template: *template,
				},
			})
		Expect(err).ToNot(HaveOccurred())

		By(fmt.Sprintf("waiting for %s to become ready", nginxTemplate))
		err = wait.PollImmediate(2*time.Second, 20*time.Minute, ready.CheckTemplateInstanceIsReady(sanity.Checker.Client.EndUser.TemplateV1.TemplateInstances(namespace), namespace))
		Expect(err).NotTo(HaveOccurred())

		By(fmt.Sprintf("getting the %s pod", nginxTemplate))
		nginxPods, err := sanity.Checker.Client.EndUser.CoreV1.Pods(namespace).List(metav1.ListOptions{
			LabelSelector: fmt.Sprintf("name=%s", nginxTemplate),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(nginxPods).NotTo(BeNil())
		Expect(len(nginxPods.Items)).To(Equal(1))

		pod := nginxPods.Items[0]
		hostIP := pod.Status.HostIP
		maxTimeSeconds := 5
		timeoutMessage := fmt.Sprintf("Connection timed out after %d seconds", maxTimeSeconds)
		checkInsecureReadOnlyPort := fmt.Sprintf("curl -sk --max-time %d http://%s:10255/metrics || echo %s", maxTimeSeconds, hostIP, timeoutMessage)
		checkSecurePort := fmt.Sprintf("curl -sk --max-time %d https://%s:10250/metrics || echo %s", maxTimeSeconds, hostIP, timeoutMessage)

		By(fmt.Sprintf("executing %q in pod running on %s", checkInsecureReadOnlyPort, hostIP))
		stdout, stderr, err := exec.RunCommandInPod(sanity.Checker.Client.Admin, &pod, checkInsecureReadOnlyPort)
		Expect(err).NotTo(HaveOccurred())
		Expect(stderr).To(BeEmpty())
		Expect(stdout).To(ContainSubstring(timeoutMessage))

		By(fmt.Sprintf("executing %q in pod running on %s", checkSecurePort, hostIP))
		stdout, stderr, err = exec.RunCommandInPod(sanity.Checker.Client.Admin, &pod, checkSecurePort)
		Expect(err).NotTo(HaveOccurred())
		Expect(stderr).To(BeEmpty())
		Expect(stdout).NotTo(ContainSubstring(timeoutMessage))
		Expect(stdout).To(ContainSubstring("Forbidden (user=system:anonymous, verb=get, resource=nodes, subresource=metrics)"))
	})

	It("should ensure no unnecessary VM rotations occured", func() {
		Expect(os.Getenv("RESOURCEGROUP")).ToNot(BeEmpty())
		ubs := updateblob.NewBlobService(azure.RPClient.BlobStorage)

		By("reading the update blob before running an update")
		before, err := ubs.Read()
		Expect(err).ToNot(HaveOccurred())

		By("ensuring the update blob has the right amount of entries")
		Expect(len(before.HostnameHashes)).To(Equal(3)) // one per master instance
		Expect(len(before.ScalesetHashes)).To(Equal(2)) // one per worker scaleset

		By("running an update")
		external, err := azure.RPClient.OpenShiftManagedClusters.Get(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(external).NotTo(BeNil())

		updated, err := azure.RPClient.OpenShiftManagedClusters.CreateOrUpdateAndWait(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), external)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.StatusCode).To(Equal(http.StatusOK))
		Expect(updated).NotTo(BeNil())

		By("reading the update blob after running an update")
		after, err := ubs.Read()
		Expect(err).ToNot(HaveOccurred())

		By("comparing the update blob before and after an update")
		Expect(reflect.DeepEqual(before, after)).To(Equal(true))
	})

	It("should be possible for an SRE to fetch the RP plugin version", func() {
		Expect(os.Getenv("RESOURCEGROUP")).ToNot(BeEmpty())
		By("Using the OSA admin client to fetch the RP plugin version")
		result, err := azure.RPClient.OpenShiftManagedClustersAdmin.GetPluginVersion(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())
		Expect(*result.PluginVersion).NotTo(BeEmpty())
		Expect(strings.HasPrefix(*result.PluginVersion, "v")).To(BeTrue())
	})
})
