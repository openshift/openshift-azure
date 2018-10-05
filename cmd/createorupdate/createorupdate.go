package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/api"
	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/plugin"
	"github.com/openshift/openshift-azure/pkg/tls"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

var logLevel = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")

// createOrUpdate simulates the RP
func createOrUpdate(ctx context.Context, oc *v20180930preview.OpenShiftManagedCluster, entry *logrus.Entry, config api.PluginConfig) (*v20180930preview.OpenShiftManagedCluster, error) {
	// instantiate the plugin
	p := plugin.NewPlugin(entry, config)

	// convert the external API manifest into the internal API representation
	log.Info("convert to internal")
	cs := api.ConvertFromV20180930preview(oc)

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
	var oldCs *api.OpenShiftManagedCluster
	if _, err := os.Stat("_data/containerservice.yaml"); err == nil {
		log.Info("read old config")
		oldCs, err = managedcluster.ReadConfig("_data/containerservice.yaml")
		if err != nil {
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
	azuredeploy, err := p.GenerateARM(ctx, cs, oldCs != nil)
	if err != nil {
		return nil, err
	}

	// write out development files
	log.Info("write helpers")
	err = writeHelpers(cs, azuredeploy)
	if err != nil {
		return nil, err
	}

	clients, err := azureclient.NewAzureClients(ctx, cs, config)
	if err != nil {
		return nil, err
	}
	err = acceptMarketplaceAgreement(ctx, cs, clients)
	if err != nil {
		return nil, err
	}

	deployer := func(ctx context.Context, azuretemplate map[string]interface{}) error {
		log.Info("applying arm template deployment")
		future, err := clients.Deployments.CreateOrUpdate(ctx, cs.Properties.AzProfile.ResourceGroup, "azuredeploy", resources.Deployment{
			Properties: &resources.DeploymentProperties{
				Template: azuretemplate,
				Mode:     resources.Incremental,
			},
		})
		if err != nil {
			return err
		}
		log.Info("waiting for arm template deployment to complete")
		return future.WaitForCompletionRef(ctx, clients.Deployments.Client)
	}

	err = p.CreateOrUpdate(ctx, cs, azuredeploy, oldCs != nil, deployer)
	if err != nil {
		return nil, err
	}

	// convert our (probably changed) internal API representation back to the
	// external API manifest to return it to the user
	oc = api.ConvertToV20180930preview(cs)

	return oc, nil
}

func acceptMarketplaceAgreement(ctx context.Context, cs *api.OpenShiftManagedCluster, clients *azureclient.AzureClients) error {
	if config.Derived.ImageResourceName() != "" ||
		os.Getenv("AUTOACCEPT_MARKETPLACE_AGREEMENT") != "yes" {
		return nil
	}

	log.Info("checking marketplace agreement")
	terms, err := clients.MarketPlaceAgreements.Get(ctx, cs.Config.ImagePublisher, cs.Config.ImageOffer, cs.Config.ImageSKU)
	if err != nil {
		return err
	}

	if *terms.AgreementProperties.Accepted {
		return nil
	}

	terms.AgreementProperties.Accepted = to.BoolPtr(true)

	log.Info("accepting marketplace agreement")
	_, err = clients.MarketPlaceAgreements.Create(ctx, cs.Config.ImagePublisher, cs.Config.ImageOffer, cs.Config.ImageSKU, terms)
	return err
}

func enrich(cs *api.OpenShiftManagedCluster) error {
	for _, env := range []string{
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_TENANT_ID",
		"DNS_DOMAIN",
		"RESOURCEGROUP",
	} {
		if os.Getenv(env) == "" {
			return fmt.Errorf("must set %s", env)
		}
	}

	cs.Properties.AzProfile = &api.AzProfile{
		TenantID:       os.Getenv("AZURE_TENANT_ID"),
		SubscriptionID: os.Getenv("AZURE_SUBSCRIPTION_ID"),
		ResourceGroup:  os.Getenv("RESOURCEGROUP"),
	}

	cs.Properties.RouterProfiles = []api.RouterProfile{
		{
			Name:            "default",
			PublicSubdomain: fmt.Sprintf("%s.%s", os.Getenv("RESOURCEGROUP"), os.Getenv("DNS_DOMAIN")),
			FQDN:            fmt.Sprintf("%s-router.%s.cloudapp.azure.com", cs.Properties.AzProfile.ResourceGroup, cs.Location),
		},
	}

	cs.Properties.ServicePrincipalProfile = &api.ServicePrincipalProfile{
		ClientID: os.Getenv("AZURE_CLIENT_ID"),
		Secret:   os.Getenv("AZURE_CLIENT_SECRET"),
	}

	return nil
}

func writeHelpers(c *api.OpenShiftManagedCluster, azuredeploy []byte) error {
	b, err := config.Derived.CloudProviderConf(c)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile("_data/_out/azure.conf", b, 0600)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile("_data/_out/azuredeploy.json", azuredeploy, 0600)
	if err != nil {
		return err
	}

	b, err = tls.PrivateKeyAsBytes(c.Config.SSHKey)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_data/_out/id_rsa", b, 0600)
	if err != nil {
		return err
	}

	b, err = yaml.Marshal(c.Config.AdminKubeconfig)
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
	ctx = context.WithValue(ctx, api.ContextKeyClientID, os.Getenv("AZURE_CLIENT_ID"))
	ctx = context.WithValue(ctx, api.ContextKeyClientSecret, os.Getenv("AZURE_CLIENT_SECRET"))
	ctx = context.WithValue(ctx, api.ContextKeyTenantID, os.Getenv("AZURE_TENANT_ID"))

	// simulate the API call to the RP
	entry := logrus.NewEntry(logger).WithFields(logrus.Fields{"resourceGroup": os.Getenv("RESOURCEGROUP")})
	var config = api.PluginConfig{SyncImage: os.Getenv("SYNC_IMAGE"),
		AcceptLanguages: []string{"en-us"}}
	oc, err = createOrUpdate(ctx, oc, entry, config)
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
