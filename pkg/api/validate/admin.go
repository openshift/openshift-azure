package validate

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/go-test/deep"

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
func (v *AdminAPIValidator) Validate(cs, oldCs *api.OpenShiftManagedCluster) (errs []error) {
	// TODO are these error messages confusing since they may not correspond with the external model?
	if cs == nil {
		errs = append(errs, fmt.Errorf("cs cannot be nil"))
		return
	}

	if oldCs == nil {
		errs = append(errs, errors.New("admin requests cannot create clusters"))
		return
	}

	errs = append(errs, validateContainerService(cs, false)...)

	errs = append(errs, v.validateUpdateContainerService(cs, oldCs)...)

	// this limits use of RunningUnderTest variable inside our validators
	// TODO: When removed this should be part of common validators
	for _, app := range cs.Properties.AgentPoolProfiles {
		errs = append(errs, validateVMSize(&app, v.runningUnderTest)...)
	}

	return
}

func (v *AdminAPIValidator) validateUpdateContainerService(cs, oldCs *api.OpenShiftManagedCluster) (errs []error) {
	if cs == nil || oldCs == nil {
		errs = append(errs, fmt.Errorf("cs and oldCs cannot be nil"))
		return
	}

	old := oldCs.DeepCopy()

	old.Config.ComponentLogLevel = cs.Config.ComponentLogLevel

	if cs.Config.PluginVersion == "latest" {
		old.Config.PluginVersion = cs.Config.PluginVersion
	}

	if !reflect.DeepEqual(cs, old) {
		errs = append(errs, fmt.Errorf("invalid change %s", deep.Equal(cs, old)))
	}

	return
}
