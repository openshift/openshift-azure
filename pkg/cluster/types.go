package cluster

//go:generate go get github.com/golang/mock/mockgen
//go:generate mockgen -destination=../util/mocks/mock_$GOPACKAGE/types.go github.com/openshift/openshift-azure/pkg/$GOPACKAGE Upgrader
//go:generate gofmt -s -l -w ../util/mocks/mock_$GOPACKAGE/types.go
//go:generate go get golang.org/x/tools/cmd/goimports
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../util/mocks/mock_$GOPACKAGE/types.go

import (
	"context"

	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/cluster/kubeclient"
	"github.com/openshift/openshift-azure/pkg/cluster/scaler"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/startup"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/compute"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/keyvault"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
	"github.com/openshift/openshift-azure/pkg/util/enrich"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

// here follow well known container and blob names
const (
	ConfigContainerName     = "config"
	SyncBlobName            = "sync"
	MasterStartupBlobName   = "master-startup"
	WorkerStartupBlobName   = "worker-startup"
	EtcdBackupContainerName = "etcd"
)

// Upgrader is the public interface to the upgrade module used by the plugin.
type Upgrader interface {
	CreateOrUpdateConfigStorageAccount(ctx context.Context) error
	EnrichCertificatesFromVault(ctx context.Context) error
	EnrichStorageAccountKeys(ctx context.Context) error
	InitializeUpdateBlob(suffix string) error
	WaitForHealthzStatusOk(ctx context.Context) error
	HealthCheck(ctx context.Context) *api.PluginError
	SortedAgentPoolProfilesForRole(role api.AgentPoolProfileRole) []api.AgentPoolProfile
	WaitForNodesInAgentPoolProfile(ctx context.Context, app *api.AgentPoolProfile, suffix string) error
	UpdateMasterAgentPool(ctx context.Context, app *api.AgentPoolProfile) *api.PluginError
	UpdateWorkerAgentPool(ctx context.Context, app *api.AgentPoolProfile, suffix string) *api.PluginError
	CreateOrUpdateSyncPod(ctx context.Context) error
	EtcdListBackups(ctx context.Context) ([]azstorage.Blob, error)
	EtcdRestoreDeleteMasterScaleSet(ctx context.Context) *api.PluginError
	EtcdRestoreDeleteMasterScaleSetHashes(ctx context.Context) *api.PluginError
	ResetUpdateBlob() error
	Reimage(ctx context.Context, scaleset, instanceID string) error
	ListVMHostnames(ctx context.Context) ([]string, error)
	RunCommand(ctx context.Context, scaleset, instanceID, command string) error
	WriteStartupBlobs() error
	GenerateARM(ctx context.Context, backupBlob string, isUpdate bool, suffix string) (map[string]interface{}, error)

	kubeclient.Interface
}

type Upgrade struct {
	kubeclient.Interface

	TestConfig        api.TestConfig
	AccountsClient    storage.AccountsClient
	StorageClient     storage.Client
	UpdateBlobService updateblob.BlobService
	Vmc               compute.VirtualMachineScaleSetVMsClient
	Ssc               compute.VirtualMachineScaleSetsClient
	Kvc               keyvault.KeyVaultClient
	Log               *logrus.Entry
	ScalerFactory     scaler.Factory
	Hasher            Hasher
	Arm               arm.Interface

	Cs *api.OpenShiftManagedCluster

	GetConsoleClient   func(cs *api.OpenShiftManagedCluster) wait.SimpleHTTPClient
	GetAPIServerClient func(cs *api.OpenShiftManagedCluster) wait.SimpleHTTPClient
}

var _ Upgrader = &Upgrade{}

// NewSimpleUpgrader creates a new upgrader instance
func NewSimpleUpgrader(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, initializeStorageClients, disableKeepAlives bool, testConfig api.TestConfig) (Upgrader, error) {
	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
	if err != nil {
		return nil, err
	}

	vaultauthorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyVaultClientAuthorizer)
	if err != nil {
		return nil, err
	}

	kubeclient, err := kubeclient.NewKubeclient(log, cs.Config.AdminKubeconfig, disableKeepAlives)
	if err != nil {
		return nil, err
	}

	arm, err := arm.New(ctx, log, cs, testConfig)
	if err != nil {
		return nil, err
	}

	u := &Upgrade{
		Interface: kubeclient,

		TestConfig:     testConfig,
		AccountsClient: storage.NewAccountsClient(ctx, log, cs.Properties.AzProfile.SubscriptionID, authorizer),
		Vmc:            compute.NewVirtualMachineScaleSetVMsClient(ctx, log, cs.Properties.AzProfile.SubscriptionID, authorizer),
		Ssc:            compute.NewVirtualMachineScaleSetsClient(ctx, log, cs.Properties.AzProfile.SubscriptionID, authorizer),
		Kvc:            keyvault.NewKeyVaultClient(ctx, log, vaultauthorizer),
		Log:            log,
		ScalerFactory:  scaler.NewFactory(),
		Hasher: &Hash{
			Log:            log,
			TestConfig:     testConfig,
			StartupFactory: startup.New,
			Arm:            arm,
		},
		Arm:                arm,
		Cs:                 cs,
		GetConsoleClient:   getConsoleClient,
		GetAPIServerClient: getAPIServerClient,
	}

	if initializeStorageClients {
		err = u.initializeStorageClients(ctx)
		if err != nil {
			return nil, err
		}
	}

	return u, nil
}

func (u *Upgrade) EnrichCertificatesFromVault(ctx context.Context) error {
	return enrich.CertificatesFromVault(ctx, u.Kvc, u.Cs)
}

func (u *Upgrade) EnrichStorageAccountKeys(ctx context.Context) error {
	return enrich.StorageAccountKeys(ctx, u.AccountsClient, u.Cs)
}

func (u *Upgrade) GenerateARM(ctx context.Context, backupBlob string, isUpdate bool, suffix string) (map[string]interface{}, error) {
	err := enrich.SASURIs(u.StorageClient, u.Cs)
	if err != nil {
		return nil, err
	}

	return u.Arm.Generate(ctx, backupBlob, isUpdate, suffix)
}
