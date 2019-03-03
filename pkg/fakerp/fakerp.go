package fakerp

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/fakerp/shared"
	"github.com/openshift/openshift-azure/pkg/plugin"
	"github.com/openshift/openshift-azure/pkg/tls"
	"github.com/openshift/openshift-azure/pkg/util/aadapp"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	utilrs "github.com/openshift/openshift-azure/pkg/util/randomstring"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
)

func GetDeployer(log *logrus.Entry, cs *api.OpenShiftManagedCluster) api.DeployFn {
	return func(ctx context.Context, azuretemplate map[string]interface{}) error {
		log.Info("applying arm template deployment")
		authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
		if err != nil {
			return err
		}

		deployments := azureclient.NewDeploymentsClient(ctx, cs.Properties.AzProfile.SubscriptionID, authorizer)
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

	template, err := GetPluginTemplate()
	if err != nil {
		return nil, err
	}

	// This should be executed only for fakeRP
	overridePluginTemplate(template)

	errs = p.ValidatePluginTemplate(ctx, template)
	if len(errs) > 0 {
		return nil, kerrors.NewAggregate(errs)
	}

	// read in the OpenShift config blob if it exists (i.e. we're updating)
	// in the update path, the RP should have access to the previous internal
	// API representation for comparison.
	if !shared.IsUpdate() {
		// If containerservice.yaml does not exist - it is Create call
		// create DNS records only on first call and when user does not supply them
		if cs.Properties.RouterProfiles[0].PublicSubdomain == "" {
			log.Info("routeprofiles[0].publicsubdomain was empty, randomizing publicSubdomain and routerCName")

			// generate a random domain string
			rDomain, err := utilrs.RandomString("abcdefghijklmnopqrstuvxyz0123456789", 20)
			if err != nil {
				return nil, err
			}
			publicSubdomain := fmt.Sprintf("%s.%s", rDomain, os.Getenv("DNS_DOMAIN"))

			rrCName, err := utilrs.RandomString("abcdefghijklmnopqrstuvwxyz0123456789", 20)
			if err != nil {
				return nil, err
			}
			// create router wildcard DNS CName
			routerCName := fmt.Sprintf("osa%s.%s.%s", rrCName, cs.Location, "cloudapp.azure.com")

			log.Info("routeprofiles[0].publicsubdomain was empty, creating dns")
			err = CreateOCPDNS(ctx, os.Getenv("AZURE_SUBSCRIPTION_ID"), os.Getenv("RESOURCEGROUP"), cs.Location, os.Getenv("DNS_RESOURCEGROUP"), os.Getenv("DNS_DOMAIN"), publicSubdomain, routerCName, os.Getenv("NOGROUPTAGS") == "true")
			if err != nil {
				return nil, err
			}

			cs.Properties.RouterProfiles = []api.RouterProfile{
				{
					Name:            "default",
					PublicSubdomain: publicSubdomain,
					FQDN:            routerCName,
				},
			}
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
	err = p.GenerateConfig(ctx, cs, template)
	if err != nil {
		return nil, err
	}

	if !shared.IsUpdate() {
		// call after GenerateConfig() as we need the CA certificate created
		authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
		if err != nil {
			return nil, err
		}

		graphauthorizer, err := azureclient.NewAuthorizerFromEnvironment(azure.PublicCloud.GraphEndpoint)
		if err != nil {
			return nil, err
		}

		vaultauthorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyVaultClientAuthorizer)
		if err != nil {
			return nil, err
		}

		vc := azureclient.NewVaultMgmtClient(ctx, os.Getenv("AZURE_SUBSCRIPTION_ID"), authorizer)
		spc := azureclient.NewServicePrincipalsClient(ctx, os.Getenv("AZURE_TENANT_ID"), graphauthorizer)
		kvc := azureclient.NewKeyVaultClient(ctx, vaultauthorizer)

		objID, err := aadapp.GetServicePrincipalObjectIDFromAppID(ctx, spc, cs.Properties.MasterServicePrincipalProfile.ClientID)
		if err != nil {
			return nil, err
		}
		vaultURL, err := createVault(ctx, vc, objID, os.Getenv("AZURE_TENANT_ID"), os.Getenv("RESOURCEGROUP"), cs.Location, vaultName(os.Getenv("RESOURCEGROUP")))
		if err != nil {
			return nil, err
		}
		err = writeTLSCertsToVault(ctx, kvc, cs, vaultURL)
		if err != nil {
			return nil, err
		}
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

	// write out development files
	log.Info("write helpers")
	err = writeHelpers(cs)
	if err != nil {
		return nil, err
	}

	err = acceptMarketplaceAgreement(ctx, log, cs, config)
	if err != nil {
		return nil, err
	}

	log.Info("plugin createorupdate")
	deployer := GetDeployer(log, cs)
	if err := p.CreateOrUpdate(ctx, cs, oldCs != nil, deployer); err != nil {
		return nil, err
	}

	return cs, nil
}

func acceptMarketplaceAgreement(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, pluginConfig *api.PluginConfig) error {
	if pluginConfig.TestConfig.ImageResourceName != "" ||
		os.Getenv("AUTOACCEPT_MARKETPLACE_AGREEMENT") != "yes" {
		return nil
	}

	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
	if err != nil {
		return err
	}

	marketPlaceAgreements := azureclient.NewMarketPlaceAgreementsClient(ctx, cs.Properties.AzProfile.SubscriptionID, authorizer)
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

	// TODO these should be different
	cs.Properties.MasterServicePrincipalProfile = api.ServicePrincipalProfile{
		ClientID: os.Getenv("AZURE_CLIENT_ID"),
		Secret:   os.Getenv("AZURE_CLIENT_SECRET"),
	}
	cs.Properties.WorkerServicePrincipalProfile = api.ServicePrincipalProfile{
		ClientID: os.Getenv("AZURE_CLIENT_ID"),
		Secret:   os.Getenv("AZURE_CLIENT_SECRET"),
	}

	// /subscriptions/{subscription}/resourcegroups/{resource_group}/providers/Microsoft.ContainerService/openshiftmanagedClusters/{cluster_name}
	cs.ID = resourceid.ResourceID(cs.Properties.AzProfile.SubscriptionID, cs.Properties.AzProfile.ResourceGroup, "Microsoft.ContainerService/openshiftmanagedClusters", cs.Name)
	return nil
}

func writeHelpers(cs *api.OpenShiftManagedCluster) error {
	dataDir, err := shared.FindDirectory(shared.DataDirectory)
	if err != nil {
		return err
	}
	// ensure both the new key and the old key are on disk so
	// you can SSH in regardless of the state of a VM after an update
	if _, err = os.Stat(filepath.Join(dataDir, "_out/id_rsa")); err == nil {
		b, err := ioutil.ReadFile(filepath.Join(dataDir, "_out/id_rsa"))
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(dataDir, "_out/id_rsa.old"), b, 0600)
		if err != nil {
			return err
		}
	}
	b, err := config.Derived.MasterCloudProviderConf(cs)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(dataDir, "_out/azure.conf"), b, 0600)
	if err != nil {
		return err
	}

	b, err = config.Derived.AadGroupSyncConf(cs)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(dataDir, "_out/aad-group-sync.yaml"), b, 0600)
	if err != nil {
		return err
	}

	b, err = tls.PrivateKeyAsBytes(cs.Config.SSHKey)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(dataDir, "_out/id_rsa"), b, 0600)
	if err != nil {
		return err
	}

	b, err = yaml.Marshal(cs.Config.AdminKubeconfig)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(dataDir, "_out/admin.kubeconfig"), b, 0600)
}
