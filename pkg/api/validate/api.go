package validate

import (
	"github.com/openshift/openshift-azure/pkg/api"
)

// Validate validates a OpenShiftManagedCluster struct
func (v *APIValidator) Validate(cs, oldCs *api.OpenShiftManagedCluster, externalOnly bool) (errs []error) {
	errs = append(errs, validateContainerService(cs, externalOnly)...)
	errs = append(errs, validateUpdateContainerService(cs, oldCs)...)
	// this limits use of RunningUnderTest variable inside our validators
	// TODO: When removed this should be part of common validators
	for _, app := range cs.Properties.AgentPoolProfiles {
		errs = append(errs, validateVMSize(app, v.runningUnderTest)...)
	}

	return errs
}
