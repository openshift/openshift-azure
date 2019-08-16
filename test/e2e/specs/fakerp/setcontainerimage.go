package fakerp

import (
	"context"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/openshift-azure/pkg/api/admin"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/sanity"
)

var _ = Describe("Change a single image to latest E2E tests [ChangeImage][LongRunning]", func() {
	var (
		ctx = context.Background()
	)
	It("should be possible for an SRE to update a single container image", func() {
		By("getting the current Webconsole image to use")
		before, err := azure.RPClient.OpenShiftManagedClustersAdmin.Get(ctx, os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(before).NotTo(BeNil())
		beforeImage := before.Config.Images.WebConsole

		By("finding a new Webconsole image to use")
		data, err := ioutil.ReadFile("../../pluginconfig/pluginconfig-311.yaml")
		Expect(err).NotTo(HaveOccurred())
		var template *pluginapi.Config
		if err := yaml.Unmarshal(data, &template); err != nil {
			Expect(err).NotTo(HaveOccurred())
		}

		// try and use the correct image for this pluginVersion, otherwise anything different
		newImage := template.Versions[template.PluginVersion].Images.WebConsole
		if newImage == *beforeImage {
			for _, ver := range template.Versions {
				if ver.Images.WebConsole != *beforeImage {
					newImage = ver.Images.WebConsole
				}
			}
		}

		By("Executing a cluster update with updated image.")
		new := admin.OpenShiftManagedCluster{
			Config: &admin.Config{
				Images: &admin.ImageConfig{
					WebConsole: to.StringPtr(newImage),
				},
			},
		}

		By("Reading the update blob before the update")
		ubs := updateblob.NewBlobService(azure.RPClient.BlobStorage)
		beforeBlob, err := ubs.Read()
		Expect(err).ToNot(HaveOccurred())
		Expect(beforeBlob).NotTo(BeNil())
		Expect(len(beforeBlob.HostnameHashes)).To(Equal(3)) // one per master instance
		Expect(len(beforeBlob.ScalesetHashes)).To(Equal(2)) // one per worker scaleset

		update, err := azure.RPClient.OpenShiftManagedClustersAdmin.CreateOrUpdate(ctx, os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), &new)
		Expect(err).NotTo(HaveOccurred())
		Expect(update).NotTo(BeNil())

		By("checking running webconsole image")
		webconsole, err := sanity.Checker.Client.Admin.AppsV1.Deployments("openshift-web-console").Get("webconsole", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(webconsole.Spec.Template.Spec.Containers[0].Image).To(Equal(template.Versions[template.PluginVersion].Images.WebConsole))

		By("Reading the update blob after the update")
		after, err := ubs.Read()
		Expect(err).ToNot(HaveOccurred())
		Expect(after).NotTo(BeNil())

		By("Verifying that the instance hashes of the update blob are identical (masters)")
		for key, val := range beforeBlob.HostnameHashes {
			Expect(after.HostnameHashes).To(HaveKey(key))
			Expect(val).To(Equal(after.HostnameHashes[key]))
		}

		By("Verifying that the scaleset hashes of the update blob are identical (workers)")
		for key, val := range beforeBlob.ScalesetHashes {
			Expect(after.ScalesetHashes).To(HaveKey(key))
			Expect(val).To(Equal(after.ScalesetHashes[key]))
		}

		By("Validating the cluster")
		errs := sanity.Checker.ValidateCluster(context.Background())
		Expect(errs).To(BeEmpty())
	})
})
