package fakerp

import (
	"context"
	"net/http"
	"os"
	"reflect"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/jsonpath"
	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/e2e/standard"
)

var _ = Describe("Openshift on Azure admin e2e tests [Fake]", func() {
	var (
		cli *standard.SanityChecker
	)

	BeforeEach(func() {
		var err error
		cli, err = standard.NewDefaultSanityChecker()
		Expect(err).NotTo(HaveOccurred())
		Expect(cli).ToNot(BeNil())
	})

	It("should run the correct image", func() {
		// e2e check should ensure that no reg-aws images are running on box
		pods, err := cli.Client.Admin.CoreV1.Pods("").List(metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())

		for _, pod := range pods.Items {
			for _, container := range pod.Spec.Containers {
				Expect(strings.HasPrefix(container.Image, "registry.reg-aws.openshift.com/")).ToNot(BeTrue())
			}
		}

		// fetch master-000000 and determine the OS type
		master0, _ := cli.Client.Admin.CoreV1.Nodes().Get("master-000000", metav1.GetOptions{})
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
		configmaps, err := cli.Client.Admin.CoreV1.ConfigMaps("openshift-node").List(metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		var nodeConfig map[string]interface{}
		for _, configmap := range configmaps.Items {
			err = yaml.Unmarshal([]byte(configmap.Data["node-config.yaml"]), &nodeConfig)
			format := jsonpath.MustCompile("$.imageConfig.format").MustGetString(nodeConfig)
			Expect(strings.HasPrefix(format, registryPrefix)).To(BeTrue())
		}
	})

	It("should ensure no unnecessary VM rotations occured", func() {
		Expect(os.Getenv("RESOURCEGROUP")).ToNot(BeEmpty())
		azurecli, err := azure.NewClientFromEnvironment(true)
		Expect(err).ToNot(HaveOccurred())

		ubs, err := updateblob.NewBlobService(azurecli.BlobStorage)
		Expect(err).ToNot(HaveOccurred())

		By("reading the update blob before running an update")
		before, err := ubs.Read()
		Expect(err).ToNot(HaveOccurred())

		By("ensuring the update blob has the right amount of entries")
		Expect(len(before.InstanceHashes)).To(BeEquivalentTo(3)) // one per master instance
		Expect(len(before.ScalesetHashes)).To(BeEquivalentTo(2)) // one per worker scaleset

		By("running an update")
		external, err := azurecli.OpenShiftManagedClusters.Get(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(external).NotTo(BeNil())
		external.Properties.ProvisioningState = nil

		updated, err := azurecli.OpenShiftManagedClusters.CreateOrUpdateAndWait(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), external)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.StatusCode).To(Equal(http.StatusOK))
		Expect(updated).NotTo(BeNil())

		By("reading the update blob after running an update")
		after, err := ubs.Read()
		Expect(err).ToNot(HaveOccurred())

		By("comparing the update blob before and after an update")
		Expect(reflect.DeepEqual(before, after)).To(Equal(true))
	})
})
