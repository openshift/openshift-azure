package fakerp

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
	"github.com/openshift/openshift-azure/test/clients/azure"
)

var _ = Describe("Key Rotation E2E tests [KeyRotation][Fake][LongRunning]", func() {
	var (
		cli        *azure.Client
		manifest   = flag.String("manifest", "../../_data/manifest.yaml", "Path to the manifest to send to the RP")
		configBlob = flag.String("configBlob", "../../_data/containerservice.yaml", "Path to the OpenShift internal config blob")
	)

	BeforeEach(func() {
		var err error
		cli, err = azure.NewClientFromEnvironment()
		Expect(err).NotTo(HaveOccurred())
	})

	It("should be possible to maintain a healthy cluster after rotating all credentials", func() {
		ctx := context.Background()
		ctx = context.WithValue(ctx, api.ContextKeyClientID, os.Getenv("AZURE_CLIENT_ID"))
		ctx = context.WithValue(ctx, api.ContextKeyClientSecret, os.Getenv("AZURE_CLIENT_SECRET"))
		ctx = context.WithValue(ctx, api.ContextKeyTenantID, os.Getenv("AZURE_TENANT_ID"))

		By("Parsing the external manifest")
		external, err := parseExternalConfig(*manifest)
		Expect(err).NotTo(HaveOccurred())
		Expect(external).NotTo(BeNil())

		By("Parsing the internal manifest containing config blob")
		internal, err := managedcluster.ReadConfig(*configBlob)
		Expect(err).NotTo(HaveOccurred())
		Expect(internal).NotTo(BeNil())

		By("Deleting all non-ca cluster certificates and credentials from the config blob...")
		mutated := deleteSecrets(internal)
		Expect(mutated).NotTo(BeNil())

		By("Running generate on the modified config blob")
		pluginConfig, err := fakerp.GetPluginConfig()
		Expect(err).NotTo(HaveOccurred())
		configGen := config.NewSimpleGenerator(pluginConfig)
		Expect(configGen).NotTo(BeNil())
		err = configGen.Generate(mutated)
		Expect(err).NotTo(HaveOccurred())

		By("Persisting the config blob containing the new certificates and credentials")
		err = saveConfig(mutated, *configBlob)
		Expect(err).NotTo(HaveOccurred())

		By("Calling update on the fake rp with the updated config blob")
		updated, err := cli.UpdateOSACluster(ctx, external, pluginConfig)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated).NotTo(BeNil())

		By("Parsing the config blob after the update")
		internalAfterUpdate, err := managedcluster.ReadConfig(*configBlob)
		Expect(err).NotTo(HaveOccurred())
		Expect(internalAfterUpdate).NotTo(BeNil())

		By("Verifying that the initial config blob does not match the one created after the update")
		configMatch := reflect.DeepEqual(internal.Config.Certificates, internalAfterUpdate.Config.Certificates)
		Expect(configMatch).To(BeFalse())

		By("Verifying that the mutated config blob matches the one created after the update")
		configMatch = reflect.DeepEqual(mutated.Config.Certificates, internalAfterUpdate.Config.Certificates)
		Expect(configMatch).To(BeTrue())
	})
})

// deleteSecrets removes all non-ca certificates, private keys and secrets from
// OpenShiftManagedCluster.Config
// NOTE: After the admin API lands, this code should move into a newly created
// function inside the plugin, which would be listed in the external plugin
// interface, which would be responsible for resetting these values and calling
// Update on the cluster. You'd call into that from here via the admin API,
// which would call that function.
func deleteSecrets(config *api.OpenShiftManagedCluster) *api.OpenShiftManagedCluster {
	configCopy := config.DeepCopy()

	By("Removing non-ca certificates and private keys from the config blob")
	ca := configCopy.Config.Certificates.Ca
	etcd := configCopy.Config.Certificates.EtcdCa
	frontproxy := configCopy.Config.Certificates.FrontProxyCa
	servicecatalog := configCopy.Config.Certificates.ServiceCatalogCa
	servicesigning := configCopy.Config.Certificates.ServiceSigningCa
	configCopy.Config.Certificates = api.CertificateConfig{}
	configCopy.Config.Certificates.Ca = ca
	configCopy.Config.Certificates.EtcdCa = etcd
	configCopy.Config.Certificates.FrontProxyCa = frontproxy
	configCopy.Config.Certificates.ServiceCatalogCa = servicecatalog
	configCopy.Config.Certificates.ServiceSigningCa = servicesigning

	By("Removing secrets from the config blob")
	configCopy.Config.SSHKey = nil
	configCopy.Config.RegistryHTTPSecret = nil
	configCopy.Config.RegistryConsoleOAuthSecret = ""
	configCopy.Config.ConsoleOAuthSecret = ""
	configCopy.Config.AlertManagerProxySessionSecret = nil
	configCopy.Config.AlertsProxySessionSecret = nil
	configCopy.Config.PrometheusProxySessionSecret = nil
	configCopy.Config.SessionSecretAuth = nil
	configCopy.Config.SessionSecretEnc = nil
	configCopy.Config.Images.GenevaImagePullSecret = nil
	return configCopy
}

// parseExternalConfig parses an external manifest located at path and returns
// an external OpenshiftManagedCluster struct
func parseExternalConfig(path string) (*v20180930preview.OpenShiftManagedCluster, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cs *v20180930preview.OpenShiftManagedCluster
	if err := yaml.Unmarshal(b, &cs); err != nil {
		return nil, err
	}
	return cs, nil
}

// saveConfig writes an internal OpenShiftManagedCluster struct as yaml content
// at path
func saveConfig(config *api.OpenShiftManagedCluster, path string) error {
	if path == "" {
		path = "."
	}
	b, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, b, 0666)
	return err
}
