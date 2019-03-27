package config

import (
	"github.com/openshift/openshift-azure/pkg/api"
)

type simpleGenerator struct {
	cs               *api.OpenShiftManagedCluster
	runningUnderTest bool
}

func New(cs *api.OpenShiftManagedCluster, runningUnderTest bool) *simpleGenerator {
	return &simpleGenerator{
		cs:               cs,
		runningUnderTest: runningUnderTest,
	}
}
