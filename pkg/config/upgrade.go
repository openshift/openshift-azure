package config

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
)

type Upgrader interface {
	Upgrade(ctx context.Context, cs *api.OpenShiftManagedCluster) error
}

type simpleUpgrader struct{}

var _ Upgrader = &simpleUpgrader{}

func NewSimpleUpgrader(entry *logrus.Entry) Upgrader {
	log.New(entry)
	return &simpleUpgrader{}
}

func (u *simpleUpgrader) Upgrade(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	return nil
}
