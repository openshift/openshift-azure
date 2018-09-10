package config

import (
	"context"

	"github.com/sirupsen/logrus"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
)

type Upgrader interface {
	Upgrade(ctx context.Context, cs *acsapi.OpenShiftManagedCluster) error
}

type simpleUpgrader struct{}

var _ Upgrader = &simpleUpgrader{}

func NewSimpleUpgrader(entry *logrus.Entry) Upgrader {
	log.New(entry)
	return &simpleUpgrader{}
}

func (u *simpleUpgrader) Upgrade(ctx context.Context, cs *acsapi.OpenShiftManagedCluster) error {
	return nil
}
