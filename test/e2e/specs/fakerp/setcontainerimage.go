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
	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/sanity"
)

var _ = Describe("Change a single image to latest E2E tests [ChangeImage][Fake][LongRunning]", func() {
	var (
		ctx = context.Background()
	)
	It("should be possible for an SRE to update a single container image", func() {
		By("Executing a cluster update with updated image.")
		data, err := ioutil.ReadFile("../../pluginconfig/pluginconfig-311.yaml")
		Expect(err).NotTo(HaveOccurred())
		var template *pluginapi.Config
		if err := yaml.Unmarshal(data, &template); err != nil {
			Expect(err).NotTo(HaveOccurred())
		}
		before := admin.OpenShiftManagedCluster{
			Config: &admin.Config{
				Images: &admin.ImageConfig{
					WebConsole: to.StringPtr(template.Versions[template.PluginVersion].Images.WebConsole),
				},
			},
		}

		update, err := azure.FakeRPClient.OpenShiftManagedClustersAdmin.CreateOrUpdate(ctx, os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), &before)
		Expect(err).NotTo(HaveOccurred())
		Expect(update).NotTo(BeNil())

		By("checking running webconsole image")
		webconsole, err := sanity.Checker.Client.Admin.AppsV1.Deployments("openshift-web-console").Get("webconsole", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(webconsole.Spec.Template.Spec.Containers[0].Image).To(Equal(template.Versions[template.PluginVersion].Images.WebConsole))

		By("Validating the cluster")
		errs := sanity.Checker.ValidateCluster(context.Background())
		Expect(errs).To(BeEmpty())
	})
})
