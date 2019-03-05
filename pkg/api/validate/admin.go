package validate

import (
	"errors"
	"fmt"

	"github.com/openshift/openshift-azure/pkg/api"
)

// AdminAPIValidator validator for external Admin API
type AdminAPIValidator struct {
	runningUnderTest bool
}

// NewAdminValidator return instance of external Admin API validator
func NewAdminValidator(runningUnderTest bool) *AdminAPIValidator {
	return &AdminAPIValidator{runningUnderTest: runningUnderTest}
}

// Validate validates a OpenShiftManagedCluster struct
func (v *AdminAPIValidator) Validate(cs, oldCs *api.OpenShiftManagedCluster, externalOnly bool) (errs []error) {
	// TODO are these error messages confusing since they may not correspond with the external model?
	if cs == nil {
		errs = append(errs, fmt.Errorf("cs cannot be nil"))
		return
	}

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
		errs = append(errs, validateVMSize(&app, v.runningUnderTest)...)
	}

	return nil
}
