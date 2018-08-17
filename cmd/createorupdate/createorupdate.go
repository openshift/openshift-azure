package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/errors"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/api/v1"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/plugin"
	"github.com/openshift/openshift-azure/pkg/tls"
)

var logLevel = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")

// createOrUpdate simulates the RP
func createOrUpdate(oc *v1.OpenShiftManagedCluster, entry *logrus.Entry) (*v1.OpenShiftManagedCluster, error) {
	// instantiate the plugin
	p := plugin.NewPlugin(entry)

	// convert the external API manifest into the internal API representation
	log.Info("convert to internal")
	cs := acsapi.ConvertV1OpenShiftManagedClusterToOpenShiftManagedCluster(oc)

	// the RP will enrich the internal API representation with data not included
	// in the original request
	// TODO(mjudeikis): choose DNS names here
	log.Info("enrich")
	err := enrich(cs)
	if err != nil {
		return nil, err
	}

	// read in the OpenShift config blob if it exists (i.e. we're updating)
	log.Info("read old config")
	var oldCsBytes []byte
	if _, err := os.Stat("_data/containerservice.yaml"); err == nil {
		oldCsBytes, err = ioutil.ReadFile("_data/containerservice.yaml")
		if err != nil {
			return nil, err
		}
	}

	// in the update path, the RP should have access to the previous internal
	// API representation for comparison.
	var oldCs *acsapi.OpenShiftManagedCluster
	if len(oldCsBytes) > 0 {
		if err := yaml.Unmarshal(oldCsBytes, &oldCs); err != nil {
			return nil, err
		}
	}

	// validate the internal API representation (with reference to the previous
	// internal API representation)
	// we set fqdn during enrichment which is slightly different than what the RP
	// will do so we are only validating once.
	errs := p.Validate(cs, oldCs, false)
	if len(errs) > 0 {
		return nil, errors.NewAggregate(errs)
	}

	// generate or update the OpenShift config blob
	err = p.GenerateConfig(cs)
	if err != nil {
		return nil, err
	}

	// persist the OpenShift container service
	log.Info("persist config")
	bytes, err := yaml.Marshal(cs)
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile("_data/containerservice.yaml", bytes, 0600)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll("_data/_out", 0777)
	if err != nil {
		return nil, err
	}

	// generate the ARM template
	azuredeploy, err := p.GenerateARM(cs)
	if err != nil {
		return nil, err
	}

	// persist the ARM template
	log.Info("write arm")
	err = ioutil.WriteFile("_data/_out/azuredeploy.json", azuredeploy, 0600)
	if err != nil {
		return nil, err
	}

	// write out development files
	log.Info("write helpers")
	err = writeHelpers(cs.Config)
	if err != nil {
		return nil, err
	}

	// convert our (probably changed) internal API representation back to the
	// external API manifest to return it to the user
	oc = acsapi.ConvertOpenShiftManagedClusterToV1OpenShiftManagedCluster(cs)

	return oc, nil
}

func enrich(cs *acsapi.OpenShiftManagedCluster) error {
	cs.Properties.AzProfile = &acsapi.AzProfile{
		TenantID:       os.Getenv("AZURE_TENANT_ID"),
		SubscriptionID: os.Getenv("AZURE_SUBSCRIPTION_ID"),
		ResourceGroup:  os.Getenv("RESOURCEGROUP"),
	}

	if cs.Properties.AzProfile.TenantID == "" {
		return fmt.Errorf("must set AZURE_TENANT_ID")
	}
	if cs.Properties.AzProfile.SubscriptionID == "" {
		return fmt.Errorf("must set AZURE_SUBSCRIPTION_ID")
	}
	if cs.Properties.AzProfile.ResourceGroup == "" {
		return fmt.Errorf("must set RESOURCEGROUP")
	}

	// TODO: MS to provide FQDN values for OCP cluster to use.
	// cs.Properties.FQDN = random-cluster-master-prefix.eastus.cloudapp.azure.com
	// cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].FQDN = random-cluster-router-prefix.eastus.cloudapp.azure.com
	// We will:
	//    extract random-cluster-master-prefix and put it into ARM to use this for public-ip
	//    extract random-cluster-router-prefix and put it into service annotation so cloud-plugin could claim DNS name when provisioning IP

	cs.Properties.FQDN = fmt.Sprintf("%s.%s.cloudapp.azure.com", cs.Properties.AzProfile.ResourceGroup, cs.Location)
	cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].FQDN = fmt.Sprintf("%s-router.%s.cloudapp.azure.com", cs.Properties.AzProfile.ResourceGroup, cs.Location)

	return nil
}

func writeHelpers(c *acsapi.Config) error {
	b, err := tls.PrivateKeyAsBytes(c.SSHKey)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_data/_out/id_rsa", b, 0600)
	if err != nil {
		return err
	}

	b, err = yaml.Marshal(c.AdminKubeconfig)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("_data/_out/admin.kubeconfig", b, 0600)
}

func main() {
	flag.Parse()
	// mock logger configuration
	logger := logrus.New()
	// sanitize input to only accept specific log levels and tolerate junk
	logger.SetLevel(log.SanitizeLogLevel(*logLevel))
	log := logrus.NewEntry(logger)

	// read in the external API manifest.
	b, err := ioutil.ReadFile("_data/manifest.yaml")
	if err != nil {
		log.Fatal(err)
	}
	var oc *v1.OpenShiftManagedCluster
	err = yaml.Unmarshal(b, &oc)
	if err != nil {
		log.Fatal(err)
	}

	// simulate the API call to the RP
	entry := logrus.NewEntry(logger).WithFields(logrus.Fields{"resourceGroup": os.Getenv("RESOURCEGROUP")})
	oc, err = createOrUpdate(oc, entry)
	if err != nil {
		log.Fatal(err)
	}

	// persist the returned (updated) external API manifest.
	b, err = yaml.Marshal(oc)
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile("_data/manifest.yaml", b, 0666)
	if err != nil {
		log.Fatal(err)
	}
}
