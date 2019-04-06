package fakerp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/names"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/derived"
	"github.com/openshift/openshift-azure/pkg/util/random"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
	"github.com/openshift/openshift-azure/pkg/util/tls"
	"github.com/openshift/openshift-azure/pkg/util/vault"
)

func debugDeployerError(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, err error, testConfig api.TestConfig) error {
	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
	if err != nil {
		return err
	}

	deploymentOperations := azureclient.NewDeploymentOperationsClient(ctx, cs.Properties.AzProfile.SubscriptionID, authorizer)

	operations, err := deploymentOperations.List(ctx, cs.Properties.AzProfile.ResourceGroup, "azuredeploy", nil)
	if err != nil {
		log.Warnf("failed to get deployment operations: %v", err)
		return err
	}

	for _, op := range operations {
		if *op.Properties.ProvisioningState == "Succeeded" {
			continue
		}

		b, _ := json.MarshalIndent(op, "", "  ")
		log.Debug(string(b))

		if testConfig.ArtifactDir != "" &&
			*op.Properties.TargetResource.ResourceType == "Microsoft.Compute/virtualMachineScaleSets" {
			s, err := newSSHer(ctx, cs)
			if err != nil {
				log.Warnf("newSSHer failed: %v", err)
				continue
			}

			for _, app := range cs.Properties.AgentPoolProfiles {
				prefix := names.GetScalesetName(&app, "")
				if !strings.HasPrefix(*op.Properties.TargetResource.ResourceName, prefix) {
					continue
				}

				for i := int64(0); i < app.Count; i++ {
					hostname := *op.Properties.TargetResource.ResourceName + fmt.Sprintf("%06s", strconv.FormatInt(i, 36))
					cli, err := s.Dial(ctx, hostname)
					if err != nil {
						log.Warnf("Dial failed: %v", err)
						continue
					}

					err = s.RunRemoteCommandAndSaveToFile(cli, "sudo journalctl", testConfig.ArtifactDir+"/"+hostname+"-early-journal")
					if err != nil {
						log.Warnf("RunRemoteCommandAndSaveToFile failed: %v", err)
						continue
					}

					err = s.RunRemoteCommandAndSaveToFile(cli, "sudo cat /var/lib/waagent/custom-script/download/1/stdout", testConfig.ArtifactDir+"/"+hostname+"-waagent-stdout")
					if err != nil {
						log.Warnf("RunRemoteCommandAndSaveToFile failed: %v", err)
						continue
					}

					err = s.RunRemoteCommandAndSaveToFile(cli, "sudo cat /var/lib/waagent/custom-script/download/1/stderr", testConfig.ArtifactDir+"/"+hostname+"-waagent-stderr")
					if err != nil {
						log.Warnf("RunRemoteCommandAndSaveToFile failed: %v", err)
						continue
					}
				}
			}
		}
	}

	return nil
}

func GetDeployer(log *logrus.Entry, cs *api.OpenShiftManagedCluster, testConfig api.TestConfig) api.DeployFn {
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

		cli := deployments.Client()
		cli.PollingDuration = 30 * time.Minute

		log.Info("waiting for arm template deployment to complete")
		err = future.WaitForCompletionRef(ctx, cli)
		if err != nil {
			log.Warnf("deployment failed: %#v", err)
			debugDeployerError(ctx, log, cs, err, testConfig)
		}
		return err
	}
}

func createOrUpdate(ctx context.Context, p api.Plugin, log *logrus.Entry, cs, oldCs *api.OpenShiftManagedCluster, isAdmin bool, testConfig api.TestConfig) (*api.OpenShiftManagedCluster, error) {
	log.Info("enrich")
	err := enrich(cs)
	if err != nil {
		return nil, err
	}

	// validate the internal API representation (with reference to the previous
	// internal API representation)
	// we set fqdn during enrichment which is slightly different than what the RP
	// will do so we are only validating once.
	var errs []error
	if isAdmin {
		errs = p.ValidateAdmin(ctx, cs, oldCs)
	} else {
		errs = p.Validate(ctx, cs, oldCs, false)
	}
	if len(errs) > 0 {
		return nil, kerrors.NewAggregate(errs)
	}

	dm, err := newDNSManager(ctx, os.Getenv("AZURE_SUBSCRIPTION_ID"), os.Getenv("DNS_RESOURCEGROUP"), os.Getenv("DNS_DOMAIN"))
	if err != nil {
		return nil, err
	}

	log.Info("setting up DNS")
	err = dm.createOrUpdateOCPDNS(ctx, cs)
	if err != nil {
		return nil, err
	}

	vm, err := newVaultManager(ctx, os.Getenv("AZURE_SUBSCRIPTION_ID"))
	if err != nil {
		return nil, err
	}

	vaultURL, _, err := vault.SplitSecretURL(cs.Properties.APICertProfile.KeyVaultSecretURL)
	if err != nil {
		return nil, err
	}

	log.Info("setting up key vault")
	err = vm.createOrUpdateVault(ctx, log, os.Getenv("AZURE_CLIENT_ID"), cs.Properties.MasterServicePrincipalProfile.ClientID, os.Getenv("AZURE_TENANT_ID"), os.Getenv("RESOURCEGROUP"), cs.Location, vaultURL)
	if err != nil {
		return nil, err
	}

	err = vm.writeTLSCertsToVault(ctx, cs, vaultURL)
	if err != nil {
		return nil, err
	}

	// generate or update the OpenShift config blob
	err = p.GenerateConfig(ctx, cs, oldCs != nil)
	if err != nil {
		return nil, err
	}

	// write out development files
	log.Info("write helpers")
	err = os.MkdirAll("_data/_out", 0777)
	if err != nil {
		return nil, err
	}

	// persist the OpenShift container service
	log.Info("persist config")
	err = writeHelpers(log, cs)
	if err != nil {
		return nil, err
	}

	log.Info("plugin createorupdate")
	deployer := GetDeployer(log, cs, testConfig)
	if err := p.CreateOrUpdate(ctx, cs, oldCs != nil, deployer); err != nil {
		return nil, err
	}

	// persist the OpenShift container service with final fields
	log.Info("persist final config")
	err = writeHelpers(log, cs)
	if err != nil {
		return nil, err
	}

	return cs, nil
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

	cs.Properties.MasterServicePrincipalProfile = api.ServicePrincipalProfile{
		ClientID: os.Getenv("AZURE_MASTER_CLIENT_ID"),
		Secret:   os.Getenv("AZURE_MASTER_CLIENT_SECRET"),
	}
	cs.Properties.WorkerServicePrincipalProfile = api.ServicePrincipalProfile{
		ClientID: os.Getenv("AZURE_WORKER_CLIENT_ID"),
		Secret:   os.Getenv("AZURE_WORKER_CLIENT_SECRET"),
	}

	// /subscriptions/{subscription}/resourcegroups/{resource_group}/providers/Microsoft.ContainerService/openshiftmanagedClusters/{cluster_name}
	cs.ID = resourceid.ResourceID(cs.Properties.AzProfile.SubscriptionID, cs.Properties.AzProfile.ResourceGroup, "Microsoft.ContainerService/openshiftmanagedClusters", cs.Name)

	if len(cs.Properties.RouterProfiles) == 0 {
		cs.Properties.RouterProfiles = []api.RouterProfile{
			{
				Name: "default",
			},
		}
	}

	var vaultURL string
	var err error
	if cs.Properties.APICertProfile.KeyVaultSecretURL != "" {
		vaultURL, _, err = vault.SplitSecretURL(cs.Properties.APICertProfile.KeyVaultSecretURL)
		if err != nil {
			return err
		}
	} else {
		vaultURL, err = random.VaultURL("kv-")
		if err != nil {
			return err
		}
	}

	cs.Properties.APICertProfile.KeyVaultSecretURL = vaultURL + "/secrets/" + vaultKeyNamePublicHostname
	cs.Properties.RouterProfiles[0].RouterCertProfile.KeyVaultSecretURL = vaultURL + "/secrets/" + vaultKeyNameRouter

	cs.Properties.PublicHostname = "openshift." + os.Getenv("RESOURCEGROUP") + "." + os.Getenv("DNS_DOMAIN")
	cs.Properties.RouterProfiles[0].PublicSubdomain = "apps." + os.Getenv("RESOURCEGROUP") + "." + os.Getenv("DNS_DOMAIN")

	if cs.Properties.FQDN == "" {
		cs.Properties.FQDN, err = random.FQDN(cs.Location+".cloudapp.azure.com", 20)
		if err != nil {
			return err
		}
	}

	if cs.Properties.RouterProfiles[0].FQDN == "" {
		cs.Properties.RouterProfiles[0].FQDN, err = random.FQDN(cs.Location+".cloudapp.azure.com", 20)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeHelpers(log *logrus.Entry, cs *api.OpenShiftManagedCluster) error {
	b, err := derived.MasterCloudProviderConf(cs, true)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_data/_out/azure.conf", b, 0600)
	if err != nil {
		return err
	}

	b, err = derived.AadGroupSyncConf(cs)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_data/_out/aad-group-sync.yaml", b, 0600)
	if err != nil {
		return err
	}

	b, err = tls.PrivateKeyAsBytes(cs.Config.SSHKey)
	if err != nil {
		return err
	}
	// ensure both the new key and the old key are on disk so
	// you can SSH in regardless of the state of a VM after an update
	if _, err = os.Stat("_data/_out/id_rsa"); err == nil {
		oldb, err := ioutil.ReadFile("_data/_out/id_rsa")
		if err != nil {
			return err
		}
		if !bytes.Equal(b, oldb) {
			err = ioutil.WriteFile("_data/_out/id_rsa.old", oldb, 0600)
			if err != nil {
				return err
			}
		}
	}
	err = ioutil.WriteFile("_data/_out/id_rsa", b, 0600)
	if err != nil {
		return err
	}

	b, err = yaml.Marshal(cs.Config.AdminKubeconfig)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_data/_out/admin.kubeconfig", b, 0600)
	if err != nil {
		return err
	}

	bytes, err := yaml.Marshal(cs)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("_data/containerservice.yaml", bytes, 0600)
}
