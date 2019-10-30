// Package api defines the external API for the plugin.
package api

import (
	"context"
)

type contextKey string

const (
	ContextKeyClientAuthorizer      contextKey = "ClientAuthorizer"
	ContextKeyVaultClientAuthorizer contextKey = "VaultClientAuthorizer"
	ContextAcceptLanguages          contextKey = "AcceptLanguages"
)

type PluginStep string

const (
	PluginStepDeploy                              PluginStep = "Deploy"
	PluginStepInitializeUpdateBlob                PluginStep = "InitializeUpdateBlob"
	PluginStepResetUpdateBlob                     PluginStep = "ResetUpdateBlob"
	PluginStepEtcdListBackups                     PluginStep = "EtcdListBackups"
	PluginStepEtcdBackup                          PluginStep = "EtcdBackup"
	PluginStepClientCreation                      PluginStep = "ClientCreation"
	PluginStepEnrichCertificatesFromVault         PluginStep = "EnrichCertificatesFromVault"
	PluginStepEnrichStorageAccountKeys            PluginStep = "EnrichStorageAccountKeys"
	PluginStepScaleSetDelete                      PluginStep = "ScaleSetDelete"
	PluginStepWriteStartupBlobs                   PluginStep = "WriteStartupBlobs"
	PluginStepCreateOrUpdateConfigStorageAccount  PluginStep = "CreateOrUpdateConfigStorageAccount"
	PluginStepGenerateARM                         PluginStep = "GenerateARM"
	PluginStepCreateSyncPod                       PluginStep = "CreateSyncPod"
	PluginStepCreateSyncPodWaitForReady           PluginStep = "CreateSyncPodWaitForReady"
	PluginStepWaitForWaitForOpenShiftAPI          PluginStep = "WaitForOpenShiftAPI"
	PluginStepWaitForNodes                        PluginStep = "WaitForNodes"
	PluginStepWaitForReadySyncPod                 PluginStep = "WaitForReadySyncPod"
	PluginStepWaitForConsoleHealth                PluginStep = "WaitForConsoleHealth"
	PluginStepUpdateMasterAgentPoolHashScaleSet   PluginStep = "UpdateMasterAgentPoolHashScaleSet"
	PluginStepUpdateMasterAgentPoolReadBlob       PluginStep = "UpdateMasterAgentPoolReadBlob"
	PluginStepUpdateMasterAgentPoolDrain          PluginStep = "UpdateMasterAgentPoolDrain"
	PluginStepUpdateMasterAgentPoolDeallocate     PluginStep = "UpdateMasterAgentPoolDeallocate"
	PluginStepUpdateMasterAgentPoolUpdateVMs      PluginStep = "UpdateMasterAgentPoolUpdateVMs"
	PluginStepUpdateMasterAgentPoolReimage        PluginStep = "UpdateMasterAgentPoolReimage"
	PluginStepUpdateMasterAgentPoolStart          PluginStep = "UpdateMasterAgentPoolStart"
	PluginStepUpdateMasterAgentPoolWaitForReady   PluginStep = "UpdateMasterAgentPoolWaitForReady"
	PluginStepUpdateMasterAgentPoolUpdateBlob     PluginStep = "UpdateMasterAgentPoolUpdateBlob"
	PluginStepUpdateWorkerAgentPoolHashScaleSet   PluginStep = "UpdateWorkerAgentPoolHashScaleSet"
	PluginStepUpdateWorkerAgentPoolListVMs        PluginStep = "UpdateWorkerAgentPoolListVMs"
	PluginStepUpdateWorkerAgentPoolListScaleSets  PluginStep = "UpdateWorkerAgentPoolListScaleSets"
	PluginStepUpdateWorkerAgentPoolReadBlob       PluginStep = "UpdateWorkerAgentPoolReadBlob"
	PluginStepUpdateWorkerAgentPoolDrain          PluginStep = "UpdateWorkerAgentPoolDrain"
	PluginStepUpdateWorkerAgentPoolCreateScaleSet PluginStep = "UpdateWorkerAgentPoolCreateScaleSet"
	PluginStepUpdateWorkerAgentPoolUpdateScaleSet PluginStep = "UpdateWorkerAgentPoolUpdateScaleSet"
	PluginStepUpdateWorkerAgentPoolDeleteScaleSet PluginStep = "UpdateWorkerAgentPoolDeleteScaleSet"
	PluginStepUpdateWorkerAgentPoolWaitForReady   PluginStep = "UpdateWorkerAgentPoolWaitForReady"
	PluginStepUpdateWorkerAgentPoolUpdateBlob     PluginStep = "UpdateWorkerAgentPoolUpdateBlob"
	PluginStepUpdateWorkerAgentPoolDeleteVM       PluginStep = "UpdateWorkerAgentPoolDeleteVM"
	PluginStepUpdateSyncPod                       PluginStep = "UpdateSyncPod"
	PluginStepInvalidateClusterSecrets            PluginStep = "InvalidateClusterSecrets"
	PluginStepInvalidateClusterCertificates       PluginStep = "InvalidateClusterCertificates"
	PluginStepRegenerateClusterSecrets            PluginStep = "RegenerateClusterSecrets"
	PluginStepCheckRefreshCluster                 PluginStep = "CheckRefreshCluster"
)

type Command string

const (
	CommandRestartNetworkManager = "RestartNetworkManager"
	CommandRestartKubelet        = "RestartKubelet"
	CommandRestartDocker         = "RestartDocker"
)

// PluginError error returned by CreateOrUpdate to specify the step that failed.
type PluginError struct {
	Err  error
	Step PluginStep
}

var _ error = &PluginError{}

func (pe *PluginError) Error() string {
	return string(pe.Step) + ": " + pe.Err.Error()
}

// DeployFn makes it possible to plug in different logic to the deploy.
// The implementor must initiate a deployment of the given template using
// mode resources.Incremental and wait for it to complete.
// DeployFn returns a string with an IP address OR FQDN for the API server.
type DeployFn func(context.Context, map[string]interface{}) (*string, error)

// TestConfig holds all testing variables. It should be the zero value in
// production.
type TestConfig struct {
	RunningUnderTest   bool
	DebugHashFunctions bool
	ImageResourceGroup string
	ImageResourceName  string
	ArtifactDir        string
}

// Plugin is the main interface to openshift-azure
type Plugin interface {
	// Validate exists (a) to be able to place validation logic in a
	// single place in the event of multiple external API versions, and (b) to
	// be able to compare a new API manifest against a pre-existing API manifest
	// (for update, upgrade, etc.)
	// externalOnly indicates that fields set by the RP (FQDN and routerProfile.FQDN)
	// should be excluded.
	Validate(ctx context.Context, new, old *OpenShiftManagedCluster, externalOnly bool) []error

	// ValidateAdmin is used for validating admin API requests.
	ValidateAdmin(ctx context.Context, new, old *OpenShiftManagedCluster) []error

	// ValidatePluginTemplate validates external config request
	ValidatePluginTemplate(ctx context.Context) []error

	// GetPrivateAPIServerIPAddress returns an IP address must be assigned to FQDN and PublicHostname when privateAPIServer is enabled.
	GetPrivateAPIServerIPAddress(cs *OpenShiftManagedCluster) (string, error)

	// GenerateConfig ensures all the necessary in-cluster config is generated
	// for an Openshift cluster.
	GenerateConfig(ctx context.Context, cs *OpenShiftManagedCluster, isUpdate bool) error

	// CreateOrUpdate either deploys or runs the update depending on the isUpdate argument
	// this will call the deployer.
	CreateOrUpdate(ctx context.Context, cs *OpenShiftManagedCluster, isUpdate bool, deployer DeployFn) *PluginError

	GenevaActions
}

// GenevaActions is the interface for all geneva actions
type GenevaActions interface {
	// ListEtcdBackups lists available etcd backup
	ListEtcdBackups(ctx context.Context, cs *OpenShiftManagedCluster) ([]GenevaActionListEtcdBackups, error)

	// RecoverEtcdCluster recovers the cluster's etcd using the backup specified in the pluginConfig
	RecoverEtcdCluster(ctx context.Context, cs *OpenShiftManagedCluster, deployer DeployFn, backupBlob string) *PluginError

	// RotateClusterSecrets rotates the secrets in a cluster's config blob and then updates the cluster
	RotateClusterSecrets(ctx context.Context, cs *OpenShiftManagedCluster, deployer DeployFn) *PluginError

	// RotateClusterCertificates rotates the certificates in a cluster's config blob and then updates the cluster
	RotateClusterCertificates(ctx context.Context, cs *OpenShiftManagedCluster, deployer DeployFn) *PluginError

	// RotateClusterCertificatesAndSecrets rotates the certificates and secrets in a cluster's config blob and then updates the cluster
	RotateClusterCertificatesAndSecrets(ctx context.Context, cs *OpenShiftManagedCluster, deployer DeployFn) *PluginError

	// GetControlPlanePods fetches a consolidated list of the control plane pods in the cluster
	GetControlPlanePods(ctx context.Context, oc *OpenShiftManagedCluster) ([]byte, error)

	// ForceUpdate forces rotates all vms in a cluster
	ForceUpdate(ctx context.Context, cs *OpenShiftManagedCluster, deployer DeployFn) *PluginError

	// ListClusterVMs returns the hostnames of all vms in a cluster
	ListClusterVMs(ctx context.Context, cs *OpenShiftManagedCluster) (*GenevaActionListClusterVMs, error)

	// Reimage reimages a virtual machine in the cluster
	Reimage(ctx context.Context, oc *OpenShiftManagedCluster, hostname string) error

	// BackupEtcdCluster backs up the cluster's etcd
	BackupEtcdCluster(ctx context.Context, cs *OpenShiftManagedCluster, backupName string) error

	// RunCommand runs a predefined command on a virtual machine in the cluster
	RunCommand(ctx context.Context, cs *OpenShiftManagedCluster, hostname string, command Command) error

	// GetPluginVersion fetches the RP plugin version
	GetPluginVersion(ctx context.Context) *GenevaActionPluginVersion
}
