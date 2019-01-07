package fakerp

import (
	"context"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
	fakerp "github.com/openshift/openshift-azure/pkg/fakerp/client"
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
		cli, err = azure.NewClientFromEnvironment(false)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should be possible to maintain a healthy cluster after rotating all credentials", func() {
		By("Parsing the internal manifest containing config blob")
		internal, err := managedcluster.ReadConfig(*configBlob)
		Expect(err).NotTo(HaveOccurred())
		Expect(internal).NotTo(BeNil())

		By("Deleting all non-ca cluster certificates and credentials from the config blob...")
		mutated := deleteSecrets(internal)
		Expect(mutated).NotTo(BeNil())

		By("Persisting the config blob containing the new certificates and credentials")
		err = saveConfig(mutated, *configBlob)
		Expect(err).NotTo(HaveOccurred())

		By("Parsing the external manifest")
		external, err := fakerp.GenerateManifestAdmin(*manifest)
		Expect(err).NotTo(HaveOccurred())
		Expect(external).NotTo(BeNil())

		By("Calling update on the fake rp with the updated config blob")
		external.Properties.ProvisioningState = nil
		updated, err := cli.OpenShiftManagedClustersAdmin.CreateOrUpdateAndWait(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), *external)
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.StatusCode).To(Equal(http.StatusOK))
		Expect(updated).NotTo(BeNil())

		By("Parsing the config blob after the update")
		internalAfterUpdate, err := managedcluster.ReadConfig(*configBlob)
		Expect(err).NotTo(HaveOccurred())
		Expect(internalAfterUpdate).NotTo(BeNil())

		By("Verifying that the initial config blob does not match the one created after the update")
		configMatch := reflect.DeepEqual(internal.Config.Certificates, updated.Config.Certificates)
		Expect(configMatch).To(BeFalse())

		By("Verifying that the updated internal config blob and the config returned after the update contain matching ca certificates")
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.ServiceSigningCa.Cert, updated.Config.Certificates.ServiceSigningCa.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.ServiceCatalogCa.Cert, updated.Config.Certificates.ServiceCatalogCa.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.FrontProxyCa.Cert, updated.Config.Certificates.FrontProxyCa.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.EtcdCa.Cert, updated.Config.Certificates.EtcdCa.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.Ca.Cert, updated.Config.Certificates.Ca.Cert)).To(BeTrue())

		By("Verifying that the updated internal config blob and the config returned after the update contain matching public certificates")
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.Admin.Cert, updated.Config.Certificates.Admin.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.AggregatorFrontProxy.Cert, updated.Config.Certificates.AggregatorFrontProxy.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.AzureClusterReader.Cert, updated.Config.Certificates.AzureClusterReader.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.EtcdClient.Cert, updated.Config.Certificates.EtcdClient.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.EtcdPeer.Cert, updated.Config.Certificates.EtcdPeer.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.EtcdServer.Cert, updated.Config.Certificates.EtcdServer.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.GenevaLogging.Cert, updated.Config.Certificates.GenevaLogging.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.GenevaMetrics.Cert, updated.Config.Certificates.GenevaMetrics.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.MasterKubeletClient.Cert, updated.Config.Certificates.MasterKubeletClient.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.MasterProxyClient.Cert, updated.Config.Certificates.MasterProxyClient.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.MasterServer.Cert, updated.Config.Certificates.MasterServer.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.NodeBootstrap.Cert, updated.Config.Certificates.NodeBootstrap.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.OpenShiftConsole.Cert, updated.Config.Certificates.OpenShiftConsole.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.OpenShiftMaster.Cert, updated.Config.Certificates.OpenShiftMaster.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.Registry.Cert, updated.Config.Certificates.Registry.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.Router.Cert, updated.Config.Certificates.Router.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.ServiceCatalogAPIClient.Cert, updated.Config.Certificates.ServiceCatalogAPIClient.Cert)).To(BeTrue())
		Expect(reflect.DeepEqual(internalAfterUpdate.Config.Certificates.ServiceCatalogServer.Cert, updated.Config.Certificates.ServiceCatalogServer.Cert)).To(BeTrue())
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
