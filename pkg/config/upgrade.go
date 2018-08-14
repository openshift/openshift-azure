package config

import (
	"github.com/sirupsen/logrus"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
)

const (
	versionLatest = 1
)

type Upgrader interface {
	Upgrade(cs *acsapi.OpenShiftManagedCluster) error
}

type simpleUpgrader struct {
	log *logrus.Entry
}

var _ Upgrader = &simpleUpgrader{}

func NewSimpleUpgrader(entry *logrus.Entry) Upgrader {
	log.New(entry)
	return &simpleUpgrader{}
}

func (u *simpleUpgrader) Upgrade(cs *acsapi.OpenShiftManagedCluster) error {
	return nil
}
