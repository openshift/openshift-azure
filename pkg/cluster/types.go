package cluster

//go:generate go get github.com/golang/mock/gomock
//go:generate go install github.com/golang/mock/mockgen
//go:generate mockgen -destination=../util/mocks/mock_$GOPACKAGE/types.go -package=mock_$GOPACKAGE -source types.go
//go:generate gofmt -s -l -w ../util/mocks/mock_$GOPACKAGE/types.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../util/mocks/mock_$GOPACKAGE/types.go

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/kubeclient"
	"github.com/openshift/openshift-azure/pkg/cluster/scaler"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

// here follow well known container and blob names
const (
	ConfigContainerName     = "config"
	ConfigBlobName          = "config"
	EtcdBackupContainerName = "etcd"
)

// Upgrader is the public interface to the upgrade module used by the plugin.
type Upgrader interface {
	CreateClients(ctx context.Context, cs *api.OpenShiftManagedCluster) error
	Initialize(ctx context.Context, cs *api.OpenShiftManagedCluster) error
	InitializeUpdateBlob(cs *api.OpenShiftManagedCluster, suffix string) error
	WaitForHealthzStatusOk(ctx context.Context, cs *api.OpenShiftManagedCluster) error
	HealthCheck(ctx context.Context, cs *api.OpenShiftManagedCluster) *api.PluginError
	SortedAgentPoolProfilesForRole(cs *api.OpenShiftManagedCluster, role api.AgentPoolProfileRole) []api.AgentPoolProfile
	WaitForNodesInAgentPoolProfile(ctx context.Context, cs *api.OpenShiftManagedCluster, app *api.AgentPoolProfile, suffix string) error
	UpdateMasterAgentPool(ctx context.Context, cs *api.OpenShiftManagedCluster, app *api.AgentPoolProfile) *api.PluginError
	UpdateWorkerAgentPool(ctx context.Context, cs *api.OpenShiftManagedCluster, app *api.AgentPoolProfile, suffix string) *api.PluginError
	EtcdRestoreDeleteMasterScaleSet(ctx context.Context, cs *api.OpenShiftManagedCluster) *api.PluginError
	EtcdRestoreDeleteMasterScaleSetHashes(ctx context.Context, cs *api.OpenShiftManagedCluster) *api.PluginError
}

type simpleUpgrader struct {
	pluginConfig      api.PluginConfig
	accountsClient    azureclient.AccountsClient
	storageClient     storage.Client
	updateBlobService updateblob.BlobService
	vmc               azureclient.VirtualMachineScaleSetVMsClient
	ssc               azureclient.VirtualMachineScaleSetsClient
	kubeclient        kubeclient.Kubeclient
	log               *logrus.Entry
	scalerFactory     scaler.Factory
	hasher            Hasher
	rt                http.RoundTripper
}

var _ Upgrader = &simpleUpgrader{}

// NewSimpleUpgrader creates a new upgrader instance
func NewSimpleUpgrader(log *logrus.Entry, pluginConfig *api.PluginConfig) Upgrader {
	return &simpleUpgrader{
		pluginConfig:  *pluginConfig,
		log:           log,
		scalerFactory: scaler.NewFactory(),
		hasher:        &hasher{pluginConfig: *pluginConfig},
	}
}

func (u *simpleUpgrader) CreateClients(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	pool := x509.NewCertPool()
	pool.AddCert(cs.Config.Certificates.Ca.Cert)

	u.rt = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: pool,
		},
	}

	authorizer, err := azureclient.GetAuthorizerFromContext(ctx)
	if err != nil {
		return err
	}
	u.accountsClient = azureclient.NewAccountsClient(ctx, cs.Properties.AzProfile.SubscriptionID, authorizer)
	u.vmc = azureclient.NewVirtualMachineScaleSetVMsClient(ctx, cs.Properties.AzProfile.SubscriptionID, authorizer)
	u.ssc = azureclient.NewVirtualMachineScaleSetsClient(ctx, cs.Properties.AzProfile.SubscriptionID, authorizer)

	u.kubeclient, err = kubeclient.NewKubeclient(u.log, cs.Config.AdminKubeconfig, &u.pluginConfig)
	return err
}
