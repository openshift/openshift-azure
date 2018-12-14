package fakerp

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/fakerp/shared"
	"github.com/openshift/openshift-azure/pkg/plugin"
	"github.com/openshift/openshift-azure/pkg/tls"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

func GetDeployer(log *logrus.Entry, cs *api.OpenShiftManagedCluster, config *api.PluginConfig) api.DeployFn {
	return func(ctx context.Context, azuretemplate map[string]interface{}) error {
		log.Info("applying arm template deployment")
		authorizer, err := azureclient.NewAuthorizerFromContext(ctx)
		if err != nil {
			return err
		}

		deployments := azureclient.NewDeploymentsClient(cs.Properties.AzProfile.SubscriptionID, authorizer, config.AcceptLanguages)
		future, err := deployments.CreateOrUpdate(ctx, cs.Properties.AzProfile.ResourceGroup, "azuredeploy", resources.Deployment{
			Properties: &resources.DeploymentProperties{
				Template: azuretemplate,
				Mode:     resources.Incremental,
			},
		})
		if err != nil {
			return err
		}

		log.Info("waiting for arm template deployment to complete")
		if err := future.WaitForCompletionRef(ctx, deployments.Client()); err != nil {
			return fmt.Errorf("failed waiting for arm template deployment to complete: %#v", err)
		}
		if _, err := future.Result(deployments.DeploymentClient()); err != nil {
			return fmt.Errorf("failed to get arm template deloyment result: %#v", err)
		}
		return nil
	}
}

func createOrUpdate(ctx context.Context, log *logrus.Entry, cs, oldCs *api.OpenShiftManagedCluster, config *api.PluginConfig, isAdmin bool) (*api.OpenShiftManagedCluster, error) {
	// instantiate the plugin
	p, errs := plugin.NewPlugin(log, config)
	if len(errs) > 0 {
		return nil, kerrors.NewAggregate(errs)
	}

	// read in the OpenShift config blob if it exists (i.e. we're updating)
	// in the update path, the RP should have access to the previous internal
	// API representation for comparison.
	if !shared.IsUpdate() {
		// If containerservice.yaml does not exist - it is Create call
		// create DNS records only on first call
		err := CreateOCPDNS(ctx, os.Getenv("AZURE_SUBSCRIPTION_ID"), os.Getenv("RESOURCEGROUP"), cs.Location, os.Getenv("DNS_RESOURCEGROUP"), os.Getenv("DNS_DOMAIN"), os.Getenv("NOGROUPTAGS") == "true")
		if err != nil {
			return nil, err
		}
		// the RP will enrich the internal API representation with data not included
		// in the original request
		log.Info("enrich")
		err = enrich(cs)
		if err != nil {
			return nil, err
		}
	}

	// validate the internal API representation (with reference to the previous
	// internal API representation)
	// we set fqdn during enrichment which is slightly different than what the RP
	// will do so we are only validating once.
	if isAdmin {
		errs = p.ValidateAdmin(ctx, cs, oldCs)
	} else {
		errs = p.Validate(ctx, cs, oldCs, false)
	}
	if len(errs) > 0 {
		return nil, kerrors.NewAggregate(errs)
	}

	// generate or update the OpenShift config blob
	err := p.GenerateConfig(ctx, cs)
	if err != nil {
		return nil, err
	}

	// persist the OpenShift container service
	log.Info("persist config")
	bytes, err := yaml.Marshal(cs)
	if err != nil {
		return nil, err
	}
	dataDir, err := shared.FindDirectory(shared.DataDirectory)
	if err != nil {
		return nil, err
	}
	err = ioutil.WriteFile(filepath.Join(dataDir, "containerservice.yaml"), bytes, 0600)
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(filepath.Join(dataDir, "_out"), 0777)
	if err != nil {
		return nil, err
	}

	// generate the ARM template
	azuretemplate, err := p.GenerateARM(ctx, cs, oldCs != nil)
	if err != nil {
		return nil, err
	}

	// write out development files
	log.Info("write helpers")
	err = writeHelpers(cs, azuretemplate)
	if err != nil {
		return nil, err
	}

	err = acceptMarketplaceAgreement(ctx, log, cs, config)
	if err != nil {
		return nil, err
	}
	deployer := GetDeployer(log, cs, config)
	if err := p.CreateOrUpdate(ctx, cs, azuretemplate, oldCs != nil, deployer); err != nil {
		return nil, err
	}

	return cs, nil
}

func acceptMarketplaceAgreement(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, pluginConfig *api.PluginConfig) error {
	if pluginConfig.TestConfig.ImageResourceName != "" ||
		os.Getenv("AUTOACCEPT_MARKETPLACE_AGREEMENT") != "yes" {
		return nil
	}

	authorizer, err := azureclient.NewAuthorizerFromContext(ctx)
	if err != nil {
		return err
	}

	marketPlaceAgreements := azureclient.NewMarketPlaceAgreementsClient(cs.Properties.AzProfile.SubscriptionID, authorizer, pluginConfig.AcceptLanguages)
	log.Info("checking marketplace agreement")
	terms, err := marketPlaceAgreements.Get(ctx, cs.Config.ImagePublisher, cs.Config.ImageOffer, cs.Config.ImageSKU)
	if err != nil {
		return err
	}

	if *terms.AgreementProperties.Accepted {
		return nil
	}

	terms.AgreementProperties.Accepted = to.BoolPtr(true)

	log.Info("accepting marketplace agreement")
	_, err = marketPlaceAgreements.Create(ctx, cs.Config.ImagePublisher, cs.Config.ImageOffer, cs.Config.ImageSKU, terms)
	return err
}

func enrich(cs *api.OpenShiftManagedCluster) error {
	// TODO: Use kelseyhightower/envconfig
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

	cs.Properties.AzProfile = api.AzProfile{
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

	cs.Properties.ServicePrincipalProfile = api.ServicePrincipalProfile{
		ClientID: os.Getenv("AZURE_CLIENT_ID"),
		Secret:   os.Getenv("AZURE_CLIENT_SECRET"),
	}

	return nil
}

func writeHelpers(c *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}) error {
	dataDir, err := shared.FindDirectory(shared.DataDirectory)
	if err != nil {
		return err
	}
	b, err := config.Derived.CloudProviderConf(c)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(dataDir, "_out/azure.conf"), b, 0600)
	if err != nil {
		return err
	}

	azuredeploy, err := json.Marshal(azuretemplate)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(dataDir, "_out/azuredeploy.json"), azuredeploy, 0600)
	if err != nil {
		return err
	}

	b, err = tls.PrivateKeyAsBytes(c.Config.SSHKey)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(dataDir, "_out/id_rsa"), b, 0600)
	if err != nil {
		return err
	}

	b, err = yaml.Marshal(c.Config.AdminKubeconfig)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(dataDir, "_out/admin.kubeconfig"), b, 0600)
}
