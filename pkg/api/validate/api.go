package validate

import (
	"fmt"

	"github.com/openshift/openshift-azure/pkg/api"
)

// APIValidator validator for external API
type APIValidator struct {
	runningUnderTest bool
}

// NewAPIValidator return instance of external API validator
func NewAPIValidator(runningUnderTest bool) *APIValidator {
	return &APIValidator{runningUnderTest: runningUnderTest}
}

// Validate validates a OpenShiftManagedCluster struct
func (v *APIValidator) Validate(cs, oldCs *api.OpenShiftManagedCluster, externalOnly bool) (errs []error) {
	if cs == nil {
		errs = append(errs, fmt.Errorf("cs cannot be nil"))
		return
	}

	errs = append(errs, validateContainerService(cs, externalOnly)...)

	if oldCs != nil {
		errs = append(errs, validateUpdateContainerService(cs, oldCs)...)
	}

	// this limits use of RunningUnderTest variable inside our validators
	// TODO: When removed this should be part of common validators
	for _, app := range cs.Properties.AgentPoolProfiles {
		errs = append(errs, validateVMSize(&app, v.runningUnderTest)...)
	}

	return
}
