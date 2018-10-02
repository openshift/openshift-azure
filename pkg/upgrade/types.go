package upgrade

//go:generate go get github.com/golang/mock/gomock
//go:generate go install github.com/golang/mock/mockgen
//go:generate mockgen -destination=../util/mocks/mock_upgrade/types.go -package=mock_upgrade -source types.go
//go:generate gofmt -s -l -w ../util/mocks/mock_upgrade/types.go
//go:generate goimports -e -w ../util/mocks/mock_upgrade/types.go

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
)

type Upgrader interface {
	InitializeCluster(ctx context.Context, cs *api.OpenShiftManagedCluster) error
	Deploy(ctx context.Context, cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}, deployFn api.DeployFn) error
	Update(ctx context.Context, cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}, deployFn api.DeployFn) error
	HealthCheck(ctx context.Context, cs *api.OpenShiftManagedCluster) error
	WaitForInfraServices(ctx context.Context, cs *api.OpenShiftManagedCluster) error
}

type simpleUpgrader struct {
	pluginConfig api.PluginConfig
}

var _ Upgrader = &simpleUpgrader{}

func NewSimpleUpgrader(entry *logrus.Entry, pluginConfig *api.PluginConfig) Upgrader {
	log.New(entry)
	return &simpleUpgrader{
		pluginConfig: *pluginConfig,
	}
}
