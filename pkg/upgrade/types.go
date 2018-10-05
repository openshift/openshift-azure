package upgrade

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
)

type Upgrader interface {
	InitializeCluster(ctx context.Context, cs *api.OpenShiftManagedCluster) error
	Deploy(ctx context.Context, cs *api.OpenShiftManagedCluster, azuredeploy []byte, deployFn api.DeployFn) error
	Update(ctx context.Context, cs *api.OpenShiftManagedCluster, azuredeploy []byte, deployFn api.DeployFn) error
	HealthCheck(ctx context.Context, cs *api.OpenShiftManagedCluster) error
	WaitForInfraServices(ctx context.Context, cs *api.OpenShiftManagedCluster) error
}

type simpleUpgrader struct {
	pluginConfig api.PluginConfig
}

var _ Upgrader = &simpleUpgrader{}

func NewSimpleUpgrader(entry *logrus.Entry, pluginConfig api.PluginConfig) Upgrader {
	log.New(entry)
	return &simpleUpgrader{
		pluginConfig: pluginConfig,
	}
}
