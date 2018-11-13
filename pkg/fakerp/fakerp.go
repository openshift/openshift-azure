package fakerp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/api"
	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/plugin"
	"github.com/openshift/openshift-azure/pkg/tls"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

// CreateOrUpdate simulates the RP
func CreateOrUpdate(ctx context.Context, oc *v20180930preview.OpenShiftManagedCluster, entry *logrus.Entry, config *api.PluginConfig) (*v20180930preview.OpenShiftManagedCluster, error) {
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
		return nil, kerrors.NewAggregate(errs)
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

	err = acceptMarketplaceAgreement(ctx, cs, config)
	if err != nil {
		return nil, err
	}

	deployer := func(ctx context.Context, azuretemplate map[string]interface{}) error {
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
			return err
		}
		resp, err := future.Result(deployments.DeploymentClient())
		if err != nil {
			return err
		}
		if *resp.Properties.ProvisioningState != "Succeeded" {
			returnErr := fmt.Sprintf("arm deployment failed (correlation id: %s)", *resp.Properties.CorrelationID)
			dopc := resources.NewDeploymentOperationsClient(cs.Properties.AzProfile.SubscriptionID)
			dopc.Authorizer = authorizer
			if op, err := dopc.Get(ctx, cs.Properties.AzProfile.ResourceGroup, *resp.Name, *resp.Properties.CorrelationID); err != nil {
				log.Warn(err.Error())
			} else {
				returnErr = fmt.Sprintf("%s - %v", returnErr, op.Properties.StatusMessage)
			}
			return errors.New(returnErr)
		}
		return nil
	}

	if err := p.CreateOrUpdate(ctx, cs, azuretemplate, oldCs != nil, deployer); err != nil {
		return nil, err
	}

	// convert our (probably changed) internal API representation back to the
	// external API manifest to return it to the user
	oc = api.ConvertToV20180930preview(cs)

	return oc, nil
}

func acceptMarketplaceAgreement(ctx context.Context, cs *api.OpenShiftManagedCluster, pluginConfig *api.PluginConfig) error {
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

func writeHelpers(c *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}) error {
	b, err := config.Derived.CloudProviderConf(c)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile("_data/_out/azure.conf", b, 0600)
	if err != nil {
		return err
	}

	azuredeploy, err := json.Marshal(azuretemplate)
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
