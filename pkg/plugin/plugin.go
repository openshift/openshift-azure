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
	"github.com/openshift/openshift-azure/pkg/cluster"
	"github.com/openshift/openshift-azure/pkg/cluster/names"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
)

type plugin struct {
	// nothing in here should be dependent on an OpenShiftManagedCluster object
	log                    *logrus.Entry
	pluginConfig           *pluginapi.Config
	testConfig             api.TestConfig
	upgraderFactory        func(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (cluster.Upgrader, error)
	configInterfaceFactory func(cs *api.OpenShiftManagedCluster) (config.Interface, error)
}

var _ api.Plugin = &plugin{}

// NewPlugin creates a new plugin instance
func NewPlugin(log *logrus.Entry, pluginConfig *pluginapi.Config, testConfig api.TestConfig) (api.Plugin, []error) {
	return &plugin{
		log:                    log,
		pluginConfig:           pluginConfig,
		testConfig:             testConfig,
		upgraderFactory:        cluster.NewSimpleUpgrader,
		configInterfaceFactory: config.New,
	}, nil
}

func (p *plugin) Validate(ctx context.Context, new, old *api.OpenShiftManagedCluster, externalOnly bool) (errs []error) {
	p.log.Info("validating internal data models")
	validator := validate.NewAPIValidator(p.testConfig.RunningUnderTest)
	errs = validator.Validate(new, old, externalOnly)

	// if this is an update and not an upgrade, check if we can service it, and
	// if not, fail early
	if old != nil && new.Config.PluginVersion != "latest" {
		_, err := p.configInterfaceFactory(new)
		if err != nil {
			errs = append(errs, fmt.Errorf(`cluster with version %q cannot be updated by resource provider with version %q`, new.Config.PluginVersion, p.pluginConfig.PluginVersion))
		}
	}

	return
}

func (p *plugin) ValidateAdmin(ctx context.Context, new, old *api.OpenShiftManagedCluster) (errs []error) {
	p.log.Info("validating internal admin data models")
	validator := validate.NewAdminValidator(p.testConfig.RunningUnderTest)
	errs = validator.Validate(new, old)

	// if this is an update and not an upgrade, check if we can service it, and
	// if not, fail early
	if old != nil && new.Config.PluginVersion != "latest" {
		_, err := p.configInterfaceFactory(new)
		if err != nil {
			errs = append(errs, fmt.Errorf(`cluster with version %q cannot be updated by resource provider with version %q`, new.Config.PluginVersion, p.pluginConfig.PluginVersion))
		}
	}

	return
}

func (p *plugin) ValidatePluginTemplate(ctx context.Context) []error {
	p.log.Info("validating external plugin api data models")
	validator := validate.NewPluginAPIValidator()
	return validator.Validate(p.pluginConfig)
}

func (p *plugin) GenerateConfig(ctx context.Context, cs *api.OpenShiftManagedCluster, isUpdate bool) error {
	if !isUpdate || cs.Config.PluginVersion == "latest" {
		cs.Config.PluginVersion = p.pluginConfig.PluginVersion
	}

	p.log.Info("generating configs")
	configInterface, err := p.configInterfaceFactory(cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepClientCreation}
	}

	err = configInterface.Generate(p.pluginConfig)
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

	p.log.Info("generating arm templates")
	azuretemplate, err := clusterUpgrader.GenerateARM(ctx, backupBlob, true, suffix)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepGenerateARM}
	}

	err = clusterUpgrader.EtcdBlobExists(ctx, backupBlob)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepCheckEtcdBlobExists}
	}

	p.log.Info("restoring the cluster")
	if err := clusterUpgrader.EtcdRestoreDeleteMasterScaleSet(ctx); err != nil {
		return err
	}
	err = deployFn(ctx, azuretemplate)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepDeploy}
	}
	if err := clusterUpgrader.EtcdRestoreDeleteMasterScaleSetHashes(ctx); err != nil {
		return err
	}
	err = clusterUpgrader.WaitForHealthzStatusOk(ctx)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForWaitForOpenShiftAPI}
	}
	for _, app := range clusterUpgrader.SortedAgentPoolProfilesForRole(api.AgentPoolProfileRoleMaster) {
		err := clusterUpgrader.WaitForNodesInAgentPoolProfile(ctx, &app, "")
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

	err = clusterUpgrader.CreateOrUpdateConfigStorageAccount(ctx)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepCreateOrUpdateConfigStorageAccount}
	}

	p.log.Info("generating arm templates")
	azuretemplate, err := clusterUpgrader.GenerateARM(ctx, "", isUpdate, suffix)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepGenerateARM}
	}

	// set VnetID based on VnetName, do this before writing the blobs so that
	// they are exactly correct
	cs.Properties.NetworkProfile.VnetID = resourceid.ResourceID(cs.Properties.AzProfile.SubscriptionID, cs.Properties.AzProfile.ResourceGroup, "Microsoft.Network/virtualNetworks", "vnet") // TODO: should be using const

	// blobs must exist before deploy
	err = clusterUpgrader.WriteStartupBlobs()
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
	err = clusterUpgrader.EnrichCertificatesFromVault(ctx)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepEnrichCertificatesFromVault}
	}

	err = clusterUpgrader.EnrichStorageAccountKeys(ctx)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepEnrichStorageAccountKeys}
	}

	if isUpdate {
		for _, app := range clusterUpgrader.SortedAgentPoolProfilesForRole(api.AgentPoolProfileRoleMaster) {
			if perr := clusterUpgrader.UpdateMasterAgentPool(ctx, &app); perr != nil {
				return perr
			}
		}
		for _, app := range clusterUpgrader.SortedAgentPoolProfilesForRole(api.AgentPoolProfileRoleInfra) {
			if perr := clusterUpgrader.UpdateWorkerAgentPool(ctx, &app, suffix); perr != nil {
				return perr
			}
		}
		for _, app := range clusterUpgrader.SortedAgentPoolProfilesForRole(api.AgentPoolProfileRoleCompute) {
			if perr := clusterUpgrader.UpdateWorkerAgentPool(ctx, &app, suffix); perr != nil {
				return perr
			}
		}
		err = clusterUpgrader.CreateOrUpdateSyncPod(ctx)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepCreateSyncPod}
		}
		err = clusterUpgrader.WaitForReadySyncPod(ctx)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepCreateSyncPodWaitForReady}
		}

	} else {
		err = clusterUpgrader.InitializeUpdateBlob(suffix)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepInitializeUpdateBlob}
		}
		err = clusterUpgrader.WaitForHealthzStatusOk(ctx)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepWaitForWaitForOpenShiftAPI}
		}
		err = clusterUpgrader.CreateOrUpdateSyncPod(ctx)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepUpdateSyncPod}
		}

		for _, app := range clusterUpgrader.SortedAgentPoolProfilesForRole(api.AgentPoolProfileRoleMaster) {
			err := clusterUpgrader.WaitForNodesInAgentPoolProfile(ctx, &app, "")
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepWaitForNodes}
			}
		}
		for _, app := range clusterUpgrader.SortedAgentPoolProfilesForRole(api.AgentPoolProfileRoleInfra) {
			err := clusterUpgrader.WaitForNodesInAgentPoolProfile(ctx, &app, suffix)
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepWaitForNodes}
			}
		}
		for _, app := range clusterUpgrader.SortedAgentPoolProfilesForRole(api.AgentPoolProfileRoleCompute) {
			err := clusterUpgrader.WaitForNodesInAgentPoolProfile(ctx, &app, suffix)
			if err != nil {
				return &api.PluginError{Err: err, Step: api.PluginStepWaitForNodes}
			}
		}
		err := clusterUpgrader.WaitForReadySyncPod(ctx)
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepWaitForReadySyncPod}
		}
	}

	// Wait for infrastructure services to be healthy
	return clusterUpgrader.HealthCheck(ctx)
}

func (p *plugin) RotateClusterSecrets(ctx context.Context, cs *api.OpenShiftManagedCluster, deployFn api.DeployFn) *api.PluginError {
	p.log.Info("invalidating non-ca certificates, private keys and secrets")
	configInterface, err := p.configInterfaceFactory(cs)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepClientCreation}
	}

	err = configInterface.InvalidateSecrets()
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepInvalidateClusterSecrets}
	}
	p.log.Info("regenerating config including private keys and secrets")
	err = p.GenerateConfig(ctx, cs, true)
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
	err = clusterUpgrader.ResetUpdateBlob()
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
	pods, err := clusterUpgrader.ListVMHostnames(ctx)
	if err != nil {
		return nil, err
	}

	return &adminapi.GenevaActionListClusterVMs{VMs: &pods}, nil
}

func (p *plugin) Reimage(ctx context.Context, cs *api.OpenShiftManagedCluster, hostname string) error {
	if !validate.IsValidAgentPoolHostname(hostname) {
		return fmt.Errorf("invalid hostname %q", hostname)
	}

	scaleset, instanceID, err := names.GetScaleSetNameAndInstanceID(hostname)
	if err != nil {
		return err
	}

	p.log.Info("creating clients")
	clusterUpgrader, err := p.upgraderFactory(ctx, p.log, cs, true, true, p.testConfig)
	if err != nil {
		return err
	}

	p.log.Infof("reimaging %s", hostname)
	err = clusterUpgrader.Reimage(ctx, scaleset, instanceID)
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

	scaleset, instanceID, err := names.GetScaleSetNameAndInstanceID(hostname)
	if err != nil {
		return err
	}

	p.log.Info("creating clients")
	clusterUpgrader, err := p.upgraderFactory(ctx, p.log, cs, true, true, p.testConfig)
	if err != nil {
		return err
	}

	p.log.Infof("running %s on %s", command, hostname)
	return clusterUpgrader.RunCommand(ctx, scaleset, instanceID, script)
}

func (p *plugin) GetPluginVersion(ctx context.Context) *adminapi.GenevaActionPluginVersion {
	return &adminapi.GenevaActionPluginVersion{
		PluginVersion: &p.pluginConfig.PluginVersion,
	}
}
