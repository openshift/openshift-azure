package validate

import (
	"errors"

	"github.com/openshift/openshift-azure/pkg/api"
)

// Validate validates a OpenShiftManagedCluster struct
func (v *AdminAPIValidator) Validate(cs, oldCs *api.OpenShiftManagedCluster, externalOnly bool) (errs []error) {
	// TODO are these error messages confusing since they may not correspond with the external model?
	if oldCs == nil {
		errs = append(errs, errors.New("admin requests cannot create clusters"))
		return errs
	}
	if errs := validateContainerService(cs, externalOnly); len(errs) > 0 {
		return errs
	}
	if errs := validateUpdateContainerService(cs, oldCs); len(errs) > 0 {
		return errs
	}
	if errs := validateUpdateConfig(&cs.Config, &oldCs.Config); len(errs) > 0 {
		return errs
	}

	for _, app := range cs.Properties.AgentPoolProfiles {
		errs = append(errs, validateVMSize(app, v.runningUnderTest)...)
	}

	return nil
}
