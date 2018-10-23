package upgrade

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
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

type Upgrader interface {
	InitializeCluster(ctx context.Context, cs *api.OpenShiftManagedCluster) error
	Deploy(ctx context.Context, cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}, deployFn api.DeployFn) error
	Update(ctx context.Context, cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}, deployFn api.DeployFn) error
	HealthCheck(ctx context.Context, cs *api.OpenShiftManagedCluster) error
	WaitForInfraServices(ctx context.Context, cs *api.OpenShiftManagedCluster) error
}

type simpleUpgrader struct {
	pluginConfig   api.PluginConfig
	accountsClient azureclient.AccountsClient
	storageClient  storage.Client
	kubeclient     kubernetes.Interface
}

var _ Upgrader = &simpleUpgrader{}

func NewSimpleUpgrader(entry *logrus.Entry, pluginConfig *api.PluginConfig) Upgrader {
	log.New(entry)
	return &simpleUpgrader{
		pluginConfig: *pluginConfig,
	}
}
