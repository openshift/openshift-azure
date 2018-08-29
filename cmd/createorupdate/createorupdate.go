package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/errors"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/plugin"
	"github.com/openshift/openshift-azure/pkg/tls"
)

var logLevel = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")

// createOrUpdate simulates the RP
func createOrUpdate(ctx context.Context, oc *v20180930preview.OpenShiftManagedCluster, entry *logrus.Entry) (*v20180930preview.OpenShiftManagedCluster, error) {
	// instantiate the plugin
	p := plugin.NewPlugin(entry)

	// convert the external API manifest into the internal API representation
	log.Info("convert to internal")
	cs := acsapi.ConvertFromV20180930preview(oc)

	// the RP will enrich the internal API representation with data not included
	// in the original request
	log.Info("enrich")
	err := enrich(cs)
	if err != nil {
		return nil, err
	}

	// read in the OpenShift config blob if it exists (i.e. we're updating)
	// in the update path, the RP should have access to the previous internal
	// API representation for comparison.
	var oldCs *acsapi.OpenShiftManagedCluster
	if _, err := os.Stat("_data/containerservice.yaml"); err == nil {
		log.Info("read old config")
		b, err := ioutil.ReadFile("_data/containerservice.yaml")
		if err != nil {
			return nil, err
		}

		if err := yaml.Unmarshal(b, &oldCs); err != nil {
			return nil, err
		}

		log.Info("merge old and new config")
		p.MergeConfig(ctx, cs, oldCs)
	}

	// validate the internal API representation (with reference to the previous
	// internal API representation)
	// we set fqdn during enrichment which is slightly different than what the RP
	// will do so we are only validating once.
	errs := p.Validate(ctx, cs, oldCs, false)
	if len(errs) > 0 {
		return nil, errors.NewAggregate(errs)
	}

	// generate or update the OpenShift config blob
	err = p.GenerateConfig(ctx, cs)
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
	azuredeploy, err := p.GenerateARM(ctx, cs)
	if err != nil {
		return nil, err
	}

	// write out development files
	log.Info("write helpers")
	err = writeHelpers(cs.Config, azuredeploy)
	if err != nil {
		return nil, err
	}

	err = deploy(ctx, cs, p, azuredeploy)
	if err != nil {
		return nil, err
	}

	if oldCs != nil {
		err = update(ctx, cs, p)
		if err != nil {
			return nil, err
		}
	}

	err = p.HealthCheck(ctx, cs)
	if err != nil {
		return nil, err
	}

	// convert our (probably changed) internal API representation back to the
	// external API manifest to return it to the user
	oc = acsapi.ConvertToV20180930preview(cs)

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

func writeHelpers(c *acsapi.Config, azuredeploy []byte) error {
	err := ioutil.WriteFile("_data/_out/azure.conf", c.CloudProviderConf, 0600)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile("_data/_out/azuredeploy.json", azuredeploy, 0600)
	if err != nil {
		return err
	}

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
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	// sanitize input to only accept specific log levels and tolerate junk
	logger.SetLevel(log.SanitizeLogLevel(*logLevel))
	log := logrus.NewEntry(logger)

	// read in the external API manifest.
	b, err := ioutil.ReadFile("_data/manifest.yaml")
	if err != nil {
		log.Fatal(err)
	}
	var oc *v20180930preview.OpenShiftManagedCluster
	err = yaml.Unmarshal(b, &oc)
	if err != nil {
		log.Fatal(err)
	}

	//simulate Context with property bag
	ctx := context.Background()
	ctx = context.WithValue(ctx, acsapi.ContextKeyClientID, os.Getenv("AZURE_CLIENT_ID"))
	ctx = context.WithValue(ctx, acsapi.ContextKeyClientSecret, os.Getenv("AZURE_CLIENT_SECRET"))
	ctx = context.WithValue(ctx, acsapi.ContextKeyTenantID, os.Getenv("AZURE_TENANT_ID"))
	ctx = context.WithValue(ctx, acsapi.ContextKeySubscriptionId, os.Getenv("AZURE_SUBSCRIPTION_ID"))
	ctx = context.WithValue(ctx, acsapi.ContextKeyResourceGroup, os.Getenv("RESOURCEGROUP"))

	// simulate the API call to the RP
	entry := logrus.NewEntry(logger).WithFields(logrus.Fields{"resourceGroup": os.Getenv("RESOURCEGROUP")})
	oc, err = createOrUpdate(ctx, oc, entry)
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
