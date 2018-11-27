package cluster

//go:generate go get github.com/golang/mock/gomock
//go:generate go install github.com/golang/mock/mockgen
//go:generate mockgen -destination=../util/mocks/mock_$GOPACKAGE/types.go -package=mock_$GOPACKAGE -source types.go
//go:generate gofmt -s -l -w ../util/mocks/mock_$GOPACKAGE/types.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../util/mocks/mock_$GOPACKAGE/types.go

import (
	"context"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

// here follow well known container and blob names
const (
	ConfigContainerName     = "config"
	ConfigBlobName          = "config"
	updateContainerName     = "update"
	updateBlobName          = "update"
	EtcdBackupContainerName = "etcd"
)

// Upgrader is the public interface to the upgrade module used by the plugin.
type Upgrader interface {
	Deploy(ctx context.Context, cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}, deployFn api.DeployFn) *api.PluginError
	Update(ctx context.Context, cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}, deployFn api.DeployFn) *api.PluginError
	HealthCheck(ctx context.Context, cs *api.OpenShiftManagedCluster) *api.PluginError
	WaitForInfraServices(ctx context.Context, cs *api.OpenShiftManagedCluster) *api.PluginError
}

type simpleUpgrader struct {
	pluginConfig   api.PluginConfig
	accountsClient azureclient.AccountsClient
	storageClient  storage.Client
	vmc            azureclient.VirtualMachineScaleSetVMsClient
	ssc            azureclient.VirtualMachineScaleSetsClient
	kubeclient     kubernetes.Interface
	log            *logrus.Entry
}

var _ Upgrader = &simpleUpgrader{}

// NewSimpleUpgrader creates a new upgrader instance
func NewSimpleUpgrader(log *logrus.Entry, pluginConfig *api.PluginConfig) Upgrader {
	return &simpleUpgrader{
		pluginConfig: *pluginConfig,
		log:          log,
	}
}

func (u *simpleUpgrader) createClients(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	authorizer, err := azureclient.NewAuthorizerFromContext(ctx)
	if err != nil {
		return err
	}
	u.accountsClient = azureclient.NewAccountsClient(cs.Properties.AzProfile.SubscriptionID, authorizer, u.pluginConfig.AcceptLanguages)
	u.vmc = azureclient.NewVirtualMachineScaleSetVMsClient(cs.Properties.AzProfile.SubscriptionID, authorizer, u.pluginConfig.AcceptLanguages)
	u.ssc = azureclient.NewVirtualMachineScaleSetsClient(cs.Properties.AzProfile.SubscriptionID, authorizer, u.pluginConfig.AcceptLanguages)

	u.kubeclient, err = managedcluster.ClientsetFromV1Config(cs.Config.AdminKubeconfig)
	return err
}
