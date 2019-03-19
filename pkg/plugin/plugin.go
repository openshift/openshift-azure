// Package plugin holds the implementation of a plugin.
package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	adminapi "github.com/openshift/openshift-azure/pkg/api/admin/api"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin/api"
	"github.com/openshift/openshift-azure/pkg/api/validate"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/cluster"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
)

type plugin struct {
	// nothing in here should be dependent on an OpenShiftManagedCluster object
	log                 *logrus.Entry
	pluginConfig        *pluginapi.Config
	testConfig          api.TestConfig
	upgraderFactory     func(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (cluster.Upgrader, error)
	armGeneratorFactory func(ctx context.Context, cs *api.OpenShiftManagedCluster, testConfig api.TestConfig) (arm.Generator, error)
	configGenerator     config.Generator
}

var _ api.Plugin = &plugin{}

// NewPlugin creates a new plugin instance
func NewPlugin(log *logrus.Entry, pluginConfig *pluginapi.Config, testConfig api.TestConfig) (api.Plugin, []error) {
	return &plugin{
		log:                 log,
		pluginConfig:        pluginConfig,
		testConfig:          testConfig,
		upgraderFactory:     cluster.NewSimpleUpgrader,
		armGeneratorFactory: arm.NewSimpleGenerator,
		configGenerator:     config.NewSimpleGenerator(testConfig.RunningUnderTest),
	}, nil
}

func (p *plugin) Validate(ctx context.Context, new, old *api.OpenShiftManagedCluster, externalOnly bool) []error {
	p.log.Info("validating internal data models")
	validator := validate.NewAPIValidator(p.testConfig.RunningUnderTest)
	return validator.Validate(new, old, externalOnly)
}

func (p *plugin) ValidateAdmin(ctx context.Context, new, old *api.OpenShiftManagedCluster) []error {
	p.log.Info("validating internal admin data models")
	validator := validate.NewAdminValidator(p.testConfig.RunningUnderTest)
	return validator.Validate(new, old)
}

func (p *plugin) ValidatePluginTemplate(ctx context.Context) []error {
	p.log.Info("validating external plugin api data models")
	validator := validate.NewPluginAPIValidator()
	return validator.Validate(p.pluginConfig)
}

func (p *plugin) GenerateConfig(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	p.log.Info("generating configs")
	// TODO should we save off the original config here and if there are any errors we can restore it?
	err := p.configGenerator.Generate(cs, p.pluginConfig)
	if err != nil {
		return err
	}
	return nil
}

func (p *plugin) RecoverEtcdCluster(ctx context.Context, cs *api.OpenShiftManagedCluster, deployFn api.DeployFn, backupBlob string) *api.PluginError {
	suffix := fmt.Sprintf("%d", time.Now().Unix())

	p.log.Info("creating clients")
	clusterUpgrader, err := p.upgraderFactory(ctx, p.log, cs, true, true, p.testConfig)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepClientCreation}
	}
	armGenerator, err := p.armGeneratorFactory(ctx, cs, p.testConfig)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepClientCreation}
	}

	p.log.Info("generating arm templates")
	azuretemplate, err := armGenerator.Generate(ctx, cs, backupBlob, true, suffix)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepGenerateARM}
	}

	err = clusterUpgrader.EtcdBlobExists(ctx, backupBlob)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepCheckEtcdBlobExists}
	}

	p.log.Info("restoring the cluster")
	if err := clusterUpgrader.EtcdRestoreDeleteMasterScaleSet(ctx, cs); err != nil {
		return err
	}
	err = deployFn(ctx, azuretemplate)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepDeploy}
	}
	if err := clusterUpgrader.EtcdRestoreDeleteMasterScaleSetHashes(ctx, cs); err != nil {
		return err
	}
	err = clusterUpgrader.WaitForHealthzStatusOk(ctx, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForWaitForOpenShiftAPI}
	}
	for _, app := range clusterUpgrader.SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleMaster) {
		err := clusterUpgrader.WaitForNodesInAgentPoolProfile(ctx, cs, &app, "")
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepWaitForNodes}
		}
	}

	p.log.Info("running CreateOrUpdate")
	if err := p.CreateOrUpdate(ctx, cs, true, deployFn); err != nil {
		return err
	}
	return nil
}

func (p *plugin) CreateOrUpdate(ctx context.Context, cs *api.OpenShiftManagedCluster, isUpdate bool, deployFn api.DeployFn) *api.PluginError {
	suffix := fmt.Sprintf("%d", time.Now().Unix())

	p.log.Info("creating clients")
	clusterUpgrader, err := p.upgraderFactory(ctx, p.log, cs, false, true, p.testConfig)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepClientCreation}
	}

	err = clusterUpgrader.CreateOrUpdateConfigStorageAccount(ctx, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepCreateOrUpdateConfigStorageAccount}
	}

	// must be done after config storage account is created
	p.log.Info("creating more clients")
	armGenerator, err := p.armGeneratorFactory(ctx, cs, p.testConfig)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepClientCreation}
	}

	p.log.Info("generating arm templates")
	azuretemplate, err := armGenerator.Generate(ctx, cs, "", isUpdate, suffix)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepGenerateARM}
	}

	// blobs must exist before deploy
	err = clusterUpgrader.WriteStartupBlobs(cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWriteStartupBlobs}
	}

	if isUpdate {
		p.log.Info("starting update")
	} else {
		p.log.Info("starting deploy")
	}

	err = deployFn(ctx, azuretemplate)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepDeploy}
	}

	// enrich is required for the hash functions which are used below
	err = clusterUpgrader.EnrichCSFromVault(ctx, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepEnrichFromVault}
	}

	err = clusterUpgrader.EnrichCSStorageAccountKeys(ctx, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepEnrichKeys}
	}

	if isUpdate {
		for _, app := range clusterUpgrader.SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleMaster) {
			if perr := clusterUpgrader.UpdateMasterAgentPool(ctx, cs, &app); perr != nil {
				return perr
			}
		}
		for _, app := range clusterUpgrader.SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleInfra) {
			if perr := clusterUpgrader.UpdateWorkerAgentPool(ctx, cs, &app, suffix); perr != nil {
				return perr
			}
		}
		for _, app := range clusterUpgrader.SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleCompute) {
			if perr := clusterUpgrader.UpdateWorkerAgentPool(ctx, cs, &app, suffix); perr != nil {
				return perr
			}
		}
		err = clusterUpgrader.CreateOrUpdateSyncPod(ctx, cs)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepCreateSyncPod}
		}
		err = clusterUpgrader.WaitForReadySyncPod(ctx)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepCreateSyncPodWaitForReady}
		}

	} else {
		err = clusterUpgrader.InitializeUpdateBlob(cs, suffix)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepInitializeUpdateBlob}
		}
		err = clusterUpgrader.WaitForHealthzStatusOk(ctx, cs)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepWaitForWaitForOpenShiftAPI}
		}
		err = clusterUpgrader.CreateOrUpdateSyncPod(ctx, cs)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateSyncPod}
		}

		for _, app := range clusterUpgrader.SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleMaster) {
			err := clusterUpgrader.WaitForNodesInAgentPoolProfile(ctx, cs, &app, "")
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepWaitForNodes}
			}
		}
		for _, app := range clusterUpgrader.SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleInfra) {
			err := clusterUpgrader.WaitForNodesInAgentPoolProfile(ctx, cs, &app, suffix)
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepWaitForNodes}
			}
		}
		for _, app := range clusterUpgrader.SortedAgentPoolProfilesForRole(cs, api.AgentPoolProfileRoleCompute) {
			err := clusterUpgrader.WaitForNodesInAgentPoolProfile(ctx, cs, &app, suffix)
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepWaitForNodes}
			}
		}
		err := clusterUpgrader.WaitForReadySyncPod(ctx)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepWaitForSyncPod}
		}
	}

	// Wait for infrastructure services to be healthy
	if err := clusterUpgrader.HealthCheck(ctx, cs); err != nil {
		return err
	}

	if cs != nil {
		// setting VnetID based on VnetName
		cs.Properties.NetworkProfile.VnetID = resourceid.ResourceID(cs.Properties.AzProfile.SubscriptionID, cs.Properties.AzProfile.ResourceGroup, "Microsoft.Network/virtualNetworks", arm.VnetName)
	}

	// explicitly return nil if all went well
	return nil
}

func (p *plugin) RotateClusterSecrets(ctx context.Context, cs *api.OpenShiftManagedCluster, deployFn api.DeployFn) *api.PluginError {
	p.log.Info("invalidating non-ca certificates, private keys and secrets")
	err := p.configGenerator.InvalidateSecrets(cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepInvalidateClusterSecrets}
	}
	p.log.Info("regenerating config including private keys and secrets")
	err = p.GenerateConfig(ctx, cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepRegenerateClusterSecrets}
	}
	p.log.Info("running CreateOrUpdate")
	if err := p.CreateOrUpdate(ctx, cs, true, deployFn); err != nil {
		return err
	}
	return nil
}

func (p *plugin) GetControlPlanePods(ctx context.Context, cs *api.OpenShiftManagedCluster) ([]byte, error) {
	p.log.Info("creating clients")
	clusterUpgrader, err := p.upgraderFactory(ctx, p.log, cs, true, true, p.testConfig)
	if err != nil {
		return nil, &api.PluginError{Err: err, Step: api.PluginStepClientCreation}
	}

	p.log.Infof("querying control plane pods")
	pods, err := clusterUpgrader.GetControlPlanePods(ctx)
	if err != nil {
		return nil, err
	}
	return json.Marshal(pods)
}

func (p *plugin) ForceUpdate(ctx context.Context, cs *api.OpenShiftManagedCluster, deployFn api.DeployFn) *api.PluginError {
	p.log.Info("creating clients")
	clusterUpgrader, err := p.upgraderFactory(ctx, p.log, cs, true, true, p.testConfig)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepClientCreation}
	}

	p.log.Info("resetting the clusters update blob")
	err = clusterUpgrader.ResetUpdateBlob(cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepResetUpdateBlob}
	}
	p.log.Info("running CreateOrUpdate")
	if err := p.CreateOrUpdate(ctx, cs, true, deployFn); err != nil {
		return err
	}
	p.log.Info("force updates successful")
	return nil
}

func (p *plugin) ListClusterVMs(ctx context.Context, cs *api.OpenShiftManagedCluster) (*adminapi.GenevaActionListClusterVMs, error) {
	p.log.Info("creating clients")
	clusterUpgrader, err := p.upgraderFactory(ctx, p.log, cs, true, true, p.testConfig)
	if err != nil {
		return nil, err
	}

	p.log.Infof("listing cluster VMs")
	pods, err := clusterUpgrader.ListVMHostnames(ctx, cs)
	if err != nil {
		return nil, err
	}

	return &adminapi.GenevaActionListClusterVMs{VMs: &pods}, nil
}

func (p *plugin) Reimage(ctx context.Context, cs *api.OpenShiftManagedCluster, hostname string) error {
	if !validate.IsValidAgentPoolHostname(hostname) {
		return fmt.Errorf("invalid hostname %q", hostname)
	}

	scaleset, instanceID, err := config.GetScaleSetNameAndInstanceID(hostname)
	if err != nil {
		return err
	}

	p.log.Info("creating clients")
	clusterUpgrader, err := p.upgraderFactory(ctx, p.log, cs, true, true, p.testConfig)
	if err != nil {
		return err
	}

	p.log.Infof("reimaging %s", hostname)
	err = clusterUpgrader.Reimage(ctx, cs, scaleset, instanceID)
	if err != nil {
		return err
	}

	// not sure if we should do the following here: if the cluster is hosed, it
	// really might not help us.
	p.log.Infof("waiting for %s to be ready", hostname)
	if strings.HasPrefix(hostname, "master-") {
		err = clusterUpgrader.WaitForReadyMaster(ctx, hostname)
	} else {
		err = clusterUpgrader.WaitForReadyWorker(ctx, hostname)
	}
	return err
}

func (p *plugin) BackupEtcdCluster(ctx context.Context, cs *api.OpenShiftManagedCluster, backupName string) error {
	if !validate.IsValidBlobContainerName(backupName) { // no valid blob name is an invalid kubernetes name
		return fmt.Errorf("invalid backup name %q", backupName)
	}

	p.log.Info("creating clients")
	clusterUpgrader, err := p.upgraderFactory(ctx, p.log, cs, true, true, p.testConfig)
	if err != nil {
		return err
	}

	p.log.Infof("backing up cluster")
	err = clusterUpgrader.BackupCluster(ctx, backupName)
	if err != nil {
		return err
	}
	return nil
}

func (p *plugin) RunCommand(ctx context.Context, cs *api.OpenShiftManagedCluster, hostname string, command api.Command) error {
	if !validate.IsValidAgentPoolHostname(hostname) {
		return fmt.Errorf("invalid hostname %q", hostname)
	}

	var script string
	switch command {
	case api.CommandRestartNetworkManager:
		script = "systemctl restart NetworkManager.service"
	case api.CommandRestartKubelet:
		script = "systemctl restart atomic-openshift-node.service"
	default:
		return fmt.Errorf("invalid command %q", command)
	}

	scaleset, instanceID, err := config.GetScaleSetNameAndInstanceID(hostname)
	if err != nil {
		return err
	}

	p.log.Info("creating clients")
	clusterUpgrader, err := p.upgraderFactory(ctx, p.log, cs, true, true, p.testConfig)
	if err != nil {
		return err
	}

	p.log.Infof("running %s on %s", command, hostname)
	return clusterUpgrader.RunCommand(ctx, cs, scaleset, instanceID, script)
}

func (p *plugin) GetPluginVersion(ctx context.Context) *adminapi.GenevaActionPluginVersion {
	return &adminapi.GenevaActionPluginVersion{
		PluginVersion: &p.pluginConfig.PluginVersion,
	}
}
