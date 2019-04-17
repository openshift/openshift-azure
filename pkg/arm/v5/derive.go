package arm

import (
	"strings"

	"github.com/openshift/openshift-azure/pkg/api"
)

type derivedType struct{}

var derived = &derivedType{}

func (derivedType) MasterLBCNamePrefix(cs *api.OpenShiftManagedCluster) string {
	return strings.Split(cs.Properties.FQDN, ".")[0]
}

func (derivedType) RouterLBCNamePrefix(cs *api.OpenShiftManagedCluster) string {
	return strings.Split(cs.Properties.RouterProfiles[0].FQDN, ".")[0]
}
