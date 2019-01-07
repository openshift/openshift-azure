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
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
	"github.com/openshift/openshift-azure/test/clients/azure"
)

var _ = Describe("Key Rotation E2E tests [KeyRotation][Fake][LongRunning]", func() {
	var (
		cli        *azure.Client
		configBlob = flag.String("configBlob", "../../_data/containerservice.yaml", "Path to the OpenShift internal config blob")
	)

	BeforeEach(func() {
		var err error
		cli, err = azure.NewClientFromEnvironment(false)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should be possible to maintain a healthy cluster after rotating all credentials", func() {
		By("Reading the cluster state")
		before, err := cli.OpenShiftManagedClustersAdmin.Get(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		Expect(err).NotTo(HaveOccurred())
		Expect(before).NotTo(BeNil())

		// TODO: Should be replaced with key rotation RPC once it's implemented
		By("Deleting all non-ca cluster certificates and credentials from the config blob...")
		internal, err := managedcluster.ReadConfig(*configBlob)
		Expect(err).NotTo(HaveOccurred())
		Expect(internal).NotTo(BeNil())
		mutated := deleteSecrets(internal)
		Expect(mutated).NotTo(BeNil())
		err = saveConfig(mutated, *configBlob)
		Expect(err).NotTo(HaveOccurred())

		// TODO: Should be replaced with key rotation RPC once it's implemented
		By("Calling update on the fake rp with the updated config blob")
		before.Properties.ProvisioningState = nil
		after, err := cli.OpenShiftManagedClustersAdmin.CreateOrUpdateAndWait(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), before)
		Expect(err).NotTo(HaveOccurred())
		Expect(after.StatusCode).To(Equal(http.StatusOK))
		Expect(after).NotTo(BeNil())

		By("Verifying that the config blob before and after key rotation does not contain matching public certificates")
		Expect(reflect.DeepEqual(before.Config.Certificates.Admin.Cert, after.Config.Certificates.Admin.Cert)).To(BeFalse())
		Expect(reflect.DeepEqual(before.Config.Certificates.AggregatorFrontProxy.Cert, after.Config.Certificates.AggregatorFrontProxy.Cert)).To(BeFalse())
		Expect(reflect.DeepEqual(before.Config.Certificates.AzureClusterReader.Cert, after.Config.Certificates.AzureClusterReader.Cert)).To(BeFalse())
		Expect(reflect.DeepEqual(before.Config.Certificates.EtcdClient.Cert, after.Config.Certificates.EtcdClient.Cert)).To(BeFalse())
		Expect(reflect.DeepEqual(before.Config.Certificates.EtcdPeer.Cert, after.Config.Certificates.EtcdPeer.Cert)).To(BeFalse())
		Expect(reflect.DeepEqual(before.Config.Certificates.EtcdServer.Cert, after.Config.Certificates.EtcdServer.Cert)).To(BeFalse())
		Expect(reflect.DeepEqual(before.Config.Certificates.GenevaLogging.Cert, after.Config.Certificates.GenevaLogging.Cert)).To(BeFalse())
		Expect(reflect.DeepEqual(before.Config.Certificates.GenevaMetrics.Cert, after.Config.Certificates.GenevaMetrics.Cert)).To(BeFalse())
		Expect(reflect.DeepEqual(before.Config.Certificates.MasterKubeletClient.Cert, after.Config.Certificates.MasterKubeletClient.Cert)).To(BeFalse())
		Expect(reflect.DeepEqual(before.Config.Certificates.MasterProxyClient.Cert, after.Config.Certificates.MasterProxyClient.Cert)).To(BeFalse())
		Expect(reflect.DeepEqual(before.Config.Certificates.MasterServer.Cert, after.Config.Certificates.MasterServer.Cert)).To(BeFalse())
		Expect(reflect.DeepEqual(before.Config.Certificates.NodeBootstrap.Cert, after.Config.Certificates.NodeBootstrap.Cert)).To(BeFalse())
		Expect(reflect.DeepEqual(before.Config.Certificates.OpenShiftConsole.Cert, after.Config.Certificates.OpenShiftConsole.Cert)).To(BeFalse())
		Expect(reflect.DeepEqual(before.Config.Certificates.OpenShiftMaster.Cert, after.Config.Certificates.OpenShiftMaster.Cert)).To(BeFalse())
		Expect(reflect.DeepEqual(before.Config.Certificates.Registry.Cert, after.Config.Certificates.Registry.Cert)).To(BeFalse())
		Expect(reflect.DeepEqual(before.Config.Certificates.Router.Cert, after.Config.Certificates.Router.Cert)).To(BeFalse())
		Expect(reflect.DeepEqual(before.Config.Certificates.ServiceCatalogAPIClient.Cert, after.Config.Certificates.ServiceCatalogAPIClient.Cert)).To(BeFalse())
		Expect(reflect.DeepEqual(before.Config.Certificates.ServiceCatalogServer.Cert, after.Config.Certificates.ServiceCatalogServer.Cert)).To(BeFalse())

		By("Verifying that only the certificates have been updated")
		before.Config.Certificates = after.Config.Certificates
		configMatch := reflect.DeepEqual(before.Config, after.Config)
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
