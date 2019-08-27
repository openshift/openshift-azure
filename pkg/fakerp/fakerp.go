package fakerp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	azresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/names"
	"github.com/openshift/openshift-azure/pkg/fakerp/arm"
	"github.com/openshift/openshift-azure/pkg/fakerp/client"
	utilarm "github.com/openshift/openshift-azure/pkg/util/arm"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/resources"
	"github.com/openshift/openshift-azure/pkg/util/enrich"
	"github.com/openshift/openshift-azure/pkg/util/random"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
	"github.com/openshift/openshift-azure/pkg/util/vault"
)

func debugDeployerError(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, err error, testConfig api.TestConfig) error {
	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
	if err != nil {
		return err
	}

	deploymentOperations := resources.NewDeploymentOperationsClient(ctx, log, cs.Properties.AzProfile.SubscriptionID, authorizer)

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
			op.Properties.TargetResource != nil &&
			*op.Properties.TargetResource.ResourceType == "Microsoft.Compute/virtualMachineScaleSets" {
			s, err := NewSSHer(ctx, log, cs)
			if err != nil {
				log.Warnf("NewSSHer failed: %v", err)
				continue
			}

			for _, app := range cs.Properties.AgentPoolProfiles {
				prefix := names.GetScalesetName(&app, "")
				if !strings.HasPrefix(*op.Properties.TargetResource.ResourceName, prefix) {
					continue
				}

				for i := int64(0); i < app.Count; i++ {
					hostname := (*op.Properties.TargetResource.ResourceName)[3:] + fmt.Sprintf("-%06s", strconv.FormatInt(i, 36))
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

// GetDeployer return deplyer function for ARM operations
func GetDeployer(log *logrus.Entry, cs *api.OpenShiftManagedCluster, conf *client.Config, testConfig api.TestConfig) api.DeployFn {
	return func(ctx context.Context, azuretemplate map[string]interface{}) (*string, error) {
		log.Info("applying arm template deployment")
		authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
		if err != nil {
			return nil, err
		}

		deployments := resources.NewDeploymentsClient(ctx, log, cs.Properties.AzProfile.SubscriptionID, authorizer)
		future, err := deployments.CreateOrUpdate(ctx, cs.Properties.AzProfile.ResourceGroup, "azuredeploy", azresources.Deployment{
			Properties: &azresources.DeploymentProperties{
				Template: azuretemplate,
				Mode:     azresources.Incremental,
			},
		})
		if err != nil {
			return nil, err
		}

		log.Info("waiting for arm template deployment to complete")
		err = future.WaitForCompletionRef(ctx, deployments.Client())
		if err != nil {
			log.Warnf("deployment failed: %#v", err)
			debugDeployerError(ctx, log, cs, err, testConfig)
		}

		// This causes fails in our tests because deployer function is not versioned.
		// TODO: Remove this when v7 is gone and PLS is default in the deployer function
		if cs.Config.PluginVersion == "v9.0" &&
			conf != nil {
			log.Info("applying PLS deployment")
			now := time.Now().Unix()
			plsTemplate, err := arm.GenerateClusterSide(ctx, cs)
			if err != nil {
				return nil, err
			}

			future, err = deployments.CreateOrUpdate(ctx, cs.Properties.AzProfile.ResourceGroup, "plsdeploy", azresources.Deployment{
				Properties: &azresources.DeploymentProperties{
					Template: plsTemplate,
					Mode:     azresources.Incremental,
				},
			})
			if err != nil {
				return nil, err
			}

			log.Info("waiting for arm template deployment to complete")
			err = future.WaitForCompletionRef(ctx, deployments.Client())
			if err != nil {
				log.Warnf("deployment failed: %#v", err)
				debugDeployerError(ctx, log, cs, err, testConfig)
			}

			log.Info("applying PE deployment")
			peTemplate, err := arm.GenerateRPSide(ctx, cs, conf, now)
			if err != nil {
				return nil, err
			}

			future, err = deployments.CreateOrUpdate(ctx, conf.ManagementResourceGroup, fmt.Sprintf("pedeploy-%d", now), azresources.Deployment{
				Properties: &azresources.DeploymentProperties{
					Template: peTemplate,
					Mode:     azresources.Incremental,
				},
			})
			if err != nil {
				return nil, err
			}

			log.Info("waiting for arm template deployment to complete")
			err = future.WaitForCompletionRef(ctx, deployments.Client())
			if err != nil {
				log.Warnf("deployment failed: %#v", err)
				debugDeployerError(ctx, log, cs, err, testConfig)
			}

			log.Info("get PE IP address")
			nm, err := newNetworkManager(ctx, log, cs.Properties.AzProfile.SubscriptionID, conf.ManagementResourceGroup)
			if err != nil {
				return nil, err
			}
			peIP, err := nm.getPrivateEndpointIP(ctx, fmt.Sprintf("%s-%s-%d", utilarm.PrivateEndpointNamePrefix, cs.Name, now))
			if err != nil {
				return nil, err
			}
			log.Debugf("PE IP Address %s ", *peIP)
			return peIP, err
		}

		return nil, err
	}
}

func parsePluginVersion(pluginVersion string) (major, minor int, err error) {
	_, err = fmt.Sscanf(pluginVersion, "v%d.%d", &major, &minor)
	return
}

func createOrUpdateWrapper(ctx context.Context, p api.Plugin, log *logrus.Entry, cs, oldCs *api.OpenShiftManagedCluster, isAdmin bool, testConfig api.TestConfig) (*api.OpenShiftManagedCluster, error) {
	isUpdate := (oldCs != nil) // this is until we have called writeHelpers()

	log.Info("enrich")
	conf, err := client.NewConfig(log, cs)
	if err != nil {
		return nil, err
	}

	err = enrichCs(cs, conf)
	if err != nil {
		return nil, err
	}

	clients, err := newClients(ctx, log, cs, testConfig, conf)
	if err != nil {
		return nil, err
	}

	log.Info("setting up service principals")
	err = clients.aadMgr.ensureApps(ctx)
	if err != nil {
		return nil, err
	}

	var errs []error
	if isAdmin {
		errs = p.ValidateAdmin(ctx, cs, oldCs)
	} else {
		errs = p.Validate(ctx, cs, oldCs, true)
	}
	if len(errs) > 0 {
		return nil, kerrors.NewAggregate(errs)
	}

	// the real RP is responsible for validating ClusterVersion and twiddling
	// PluginVersion; this is our fake equivalent
	switch {
	case cs.Properties.ClusterVersion == "":
	case isAdmin && cs.Properties.ClusterVersion == "latest":
		oldMajor, _, err := parsePluginVersion(cs.Config.PluginVersion)
		if err != nil {
			return nil, err
		}
		if oldMajor < 3 {
			return nil, fmt.Errorf("tried to upgrade a cluster that is too old")
		}
		cs.Properties.ClusterVersion = ""
		cs.Config.PluginVersion = "latest"
	}

	log.Info("setting up DNS")
	err = clients.dnsMgr.createOrUpdateDns(ctx, cs)
	if err != nil {
		return nil, err
	}

	vaultURL, _, err := vault.SplitSecretURL(cs.Properties.APICertProfile.KeyVaultSecretURL)
	if err != nil {
		return nil, err
	}

	log.Info("setting up key vault")
	err = clients.vaultMgr.createOrUpdateVault(ctx, log, conf.ClientID, cs.Properties.MasterServicePrincipalProfile.ClientID, conf.TenantID, conf.ResourceGroup, cs.Location, vaultURL)
	if err != nil {
		return nil, err
	}

	err = clients.vaultMgr.writeTLSCertsToVault(ctx, cs, vaultURL)
	if err != nil {
		return nil, err
	}

	if cs.Properties.MonitorProfile.WorkspaceResourceID != "" {
		log.Info("enabling ContainerInsights solution on the workspace")
		err = createOrUpdateContainerInsights(ctx, log, cs)
		if err != nil {
			return nil, err
		}
		err = enrich.MonitorIDAndKey(ctx, clients.workspacesClient, cs)
		if err != nil {
			return nil, err
		}
	}

	if isAdmin {
		// This usage is incorrect and is effectively almost but not quite a
		// no-op, but it mirrors what the real RP actually does.  In reality we
		// would expect the real RP not to call ValidateAdmin() again, or to set
		// the third argument here (oldCs) to nil (since it doesn't have access
		// to oldCs any more), but as our admin validation code doesn't support
		// a nil oldCs on ValidateAdmin, the RP sets it to cs.
		errs = p.ValidateAdmin(ctx, cs, cs)
	} else {
		// This call is not a no-op: it additionally validates all the fields
		// dynamically created by the RP (azmosa.io dns, etc.).  To be true to
		// the real RP, we pass nil here instead of oldCs, because the real RP
		// doesn't have access to oldCs on the backend.
		errs = p.Validate(ctx, cs, nil, false)
	}
	if len(errs) > 0 {
		return nil, kerrors.NewAggregate(errs)
	}

	// generate or update the OpenShift config blob
	err = p.GenerateConfig(ctx, cs, isUpdate)
	if err != nil {
		return nil, err
	}

	// write out development files
	log.Info("write helpers")
	err = os.MkdirAll("_data/_out", 0750)
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
	deployer := GetDeployer(log, cs, conf, testConfig)
	if err := p.CreateOrUpdate(ctx, cs, isUpdate, deployer); err != nil {
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

func enrichCs(cs *api.OpenShiftManagedCluster, conf *client.Config) error {
	cs.Properties.AzProfile = api.AzProfile{
		TenantID:       conf.TenantID,
		SubscriptionID: conf.SubscriptionID,
		ResourceGroup:  conf.ResourceGroup,
	}

	// /subscriptions/{subscription}/resourcegroups/{resource_group}/providers/Microsoft.ContainerService/openshiftmanagedClusters/{cluster_name}
	cs.ID = resourceid.ResourceID(cs.Properties.AzProfile.SubscriptionID, cs.Properties.AzProfile.ResourceGroup, "Microsoft.ContainerService/openshiftmanagedClusters", cs.Name)

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

	cs.Properties.PublicHostname = "openshift." + conf.ResourceGroup + "." + conf.DNSDomain
	cs.Properties.RouterProfiles[0].PublicSubdomain = "apps." + conf.ResourceGroup + "." + conf.DNSDomain

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
