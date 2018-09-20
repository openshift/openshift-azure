package upgrade

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/initialize"
	"github.com/openshift/openshift-azure/pkg/log"
)

type Upgrader interface {
	initialize.Initializer
	Update(ctx context.Context, cs *api.OpenShiftManagedCluster, azuredeploy []byte, config api.PluginConfig) error
}

type simpleUpgrader struct {
	initialize.Initializer
}

var _ Upgrader = &simpleUpgrader{}

func NewSimpleUpgrader(entry *logrus.Entry, pluginConfig api.PluginConfig) Upgrader {
	log.New(entry)
	return &simpleUpgrader{
		Initializer: initialize.NewSimpleInitializer(entry, pluginConfig),
	}
}
