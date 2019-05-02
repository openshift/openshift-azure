package config

import (
	"github.com/openshift/openshift-azure/pkg/api"
)

type simpleGenerator struct {
	cs *api.OpenShiftManagedCluster
}

func New(cs *api.OpenShiftManagedCluster) *simpleGenerator {
	return &simpleGenerator{cs: cs}
}
