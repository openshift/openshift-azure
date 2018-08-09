package config

import (
	"github.com/sirupsen/logrus"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
)

const (
	versionLatest = 1
)

type Upgrader interface {
	Upgrade(cs *acsapi.ContainerService) error
}

type simpleUpgrader struct {
	log *logrus.Entry
}

var _ Upgrader = &simpleUpgrader{}

func NewSimpleUpgrader(log *logrus.Entry) Upgrader {
	return &simpleUpgrader{
		log: log,
	}
}

func (u *simpleUpgrader) Upgrade(cs *acsapi.ContainerService) error {
	return nil
}
