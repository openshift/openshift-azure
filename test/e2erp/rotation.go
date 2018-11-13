package e2erp

import (
	"context"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"syscall"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/api"
	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/fakerp"
)

// parseExternalConfig parses an external manifest located at path and returns an external OpenshiftManagedCluster struct
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

// parseInternalConfig parses an internal manifest located at path and returns an internal OpenshiftManagedCluster struct
func parseInternalConfig(path string) (*api.OpenShiftManagedCluster, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cs *api.OpenShiftManagedCluster
	if err := yaml.Unmarshal(b, &cs); err != nil {
		return nil, err
	}
	return cs, nil
}

// saveConfig marshall an internal OpenshiftManagedCluster struct to yaml and writes it to a file at path
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

// deleteCredentials deletes all non-ca certificates and credentials from an internal OpenshiftManagedCluster config blob
func deleteCertificates(config *api.OpenShiftManagedCluster) *api.OpenShiftManagedCluster {
	configCopy := config.DeepCopy()
	if configCopy.Config == nil {
		return configCopy
	}
	// remove non-ca certificates and keys
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
	return configCopy
}

//
func generateInternalConfig(internal *api.OpenShiftManagedCluster, pluginConfig *api.PluginConfig) error {
	configGen := config.NewSimpleGenerator(pluginConfig)
	return configGen.Generate(internal)
}

// newPluginConfig creates a new PluginConfig from the current environment
func newPluginConfig() *api.PluginConfig {
	tc := api.TestConfig{
		RunningUnderTest:   os.Getenv("RUNNING_UNDER_TEST") != "",
		ImageResourceGroup: os.Getenv("IMAGE_RESOURCEGROUP"),
		ImageResourceName:  os.Getenv("IMAGE_RESOURCENAME"),
		DeployOS:           os.Getenv("DEPLOY_OS"),
		ImageOffer:         os.Getenv("IMAGE_OFFER"),
		ImageVersion:       os.Getenv("IMAGE_VERSION"),
		ORegURL:            os.Getenv("OREG_URL"),
	}

	pluginConfig := &api.PluginConfig{
		SyncImage:       os.Getenv("SYNC_IMAGE"),
		LogBridgeImage:  os.Getenv("LOGBRIDGE_IMAGE"),
		AcceptLanguages: []string{"en-us"},
		TestConfig:      tc,
	}
	return pluginConfig
}

// updateCluster updates an OpenshiftManagedCluster by sending both the current external manifest and internal manifest
// which is to be used for the update
func updateCluster(ctx context.Context, external *v20180930preview.OpenShiftManagedCluster, configBlobPath string, logger *logrus.Entry, config *api.PluginConfig) (*v20180930preview.OpenShiftManagedCluster, error) {
	// Remove the provisioning state before updating
	external.Properties.ProvisioningState = ""

	var oc *v20180930preview.OpenShiftManagedCluster
	var err error
	// simulate the API call to the RP
	if err := wait.PollImmediate(5*time.Second, 1*time.Hour, func() (bool, error) {
		if oc, err = fakerp.CreateOrUpdate(ctx, external, configBlobPath, logger, config); err != nil {
			if autoRestErr, ok := err.(autorest.DetailedError); ok {
				if urlErr, ok := autoRestErr.Original.(*url.Error); ok {
					if netErr, ok := urlErr.Err.(*net.OpError); ok {
						if sysErr, ok := netErr.Err.(*os.SyscallError); ok {
							if sysErr.Err == syscall.ECONNREFUSED {
								return false, nil
							}
						}
					}
				}
			}
			return false, err
		}
		return true, nil
	}); err != nil {
		return nil, err
	}
	return oc, nil
}
