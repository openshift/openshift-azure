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
	Update(ctx context.Context, cs, oldCs *api.OpenShiftManagedCluster, azuredeploy []byte) error
}

type simpleUpgrader struct {
	initialize.Initializer
}

var _ Upgrader = &simpleUpgrader{}

func NewSimpleUpgrader(entry *logrus.Entry) Upgrader {
	log.New(entry)
	return &simpleUpgrader{
		Initializer: initialize.NewSimpleInitializer(entry),
	}
}
