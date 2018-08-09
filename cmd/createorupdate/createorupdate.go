package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/api"
	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/api/v1"
	"github.com/openshift/openshift-azure/pkg/plugin"
	"github.com/openshift/openshift-azure/pkg/tls"
)

// createOrUpdate simulates the RP
func createOrUpdate(oc *v1.OpenShiftCluster) (*v1.OpenShiftCluster, error) {
	// instantiate the plugin
	var p api.Plugin = &plugin.Plugin{}

	// validate the external API manifest
	errs := p.ValidateExternal(oc)
	if len(errs) > 0 {
		return nil, errors.NewAggregate(errs)
	}

	// convert the external API manifest into the internal API representation
	cs := acsapi.ConvertVLabsOpenShiftClusterToContainerService(oc)

	// the RP will enrich the internal API representation with data not included
	// in the original request
	// TODO(mjudeikis): choose DNS names here
	err := enrich(cs)
	if err != nil {
		return nil, err
	}

	// read in the OpenShift config blob if it exists (i.e. we're updating)
	var oldCsBytes []byte
	if _, err := os.Stat("_data/containerservice.yaml"); err == nil {
		oldCsBytes, err = ioutil.ReadFile("_data/containerservice.yaml")
		if err != nil {
			return nil, err
		}
	}

	// in the update path, the RP should have access to the previous internal
	// API representation for comparison.
	var oldCs *acsapi.ContainerService
	if len(oldCsBytes) > 0 {
		if err := yaml.Unmarshal(oldCsBytes, &oldCs); err != nil {
			return nil, err
		}
	}

	// validate the internal API representation (with reference to the previous
	// internal API representation)
	errs = p.ValidateInternal(cs, oldCs)
	if len(errs) > 0 {
		return nil, errors.NewAggregate(errs)
	}

	// generate or update the OpenShift config blob
	err = p.GenerateConfig(cs)
	if err != nil {
		return nil, err
	}

	// persist the OpenShift container service
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
	err = ioutil.WriteFile("_data/_out/azuredeploy.json", azuredeploy, 0600)
	if err != nil {
		return nil, err
	}

	// write out development files
	err = writeHelpers(cs.Config)
	if err != nil {
		return nil, err
	}

	// convert our (probably changed) internal API representation back to the
	// external API manifest to return it to the user
	oc = acsapi.ConvertContainerServiceToVLabsOpenShiftCluster(cs)

	return oc, nil
}

func enrich(cs *acsapi.ContainerService) error {
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

	cs.Properties.MasterProfile.FQDN = fmt.Sprintf("%s.%s.cloudapp.azure.com", cs.Properties.AzProfile.ResourceGroup, cs.Location)
	cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].FQDN = fmt.Sprintf("%s-router.%s.cloudapp.azure.com", cs.Properties.AzProfile.ResourceGroup, cs.Location)

	if cs.Properties.OrchestratorProfile.OpenShiftConfig.PublicHostname == "" || strings.HasSuffix(cs.Properties.OrchestratorProfile.OpenShiftConfig.PublicHostname, fmt.Sprintf("%s.cloudapp.azure.com", cs.Location)) {
		cs.Properties.OrchestratorProfile.OpenShiftConfig.PublicHostname = cs.Properties.MasterProfile.FQDN
	}
	if cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].PublicSubdomain == "" {
		cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].PublicSubdomain = cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].FQDN
	}

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
	// read in the external API manifest.
	b, err := ioutil.ReadFile("_data/manifest.yaml")
	if err != nil {
		panic(err)
	}
	var oc *v1.OpenShiftCluster
	err = yaml.Unmarshal(b, &oc)
	if err != nil {
		panic(err)
	}

	// simulate the API call to the RP
	oc, err = createOrUpdate(oc)
	if err != nil {
		panic(err)
	}

	// persist the returned (updated) external API manifest.
	b, err = yaml.Marshal(oc)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile("_data/manifest.yaml", b, 0666)
	if err != nil {
		panic(err)
	}
}
