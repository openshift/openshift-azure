package validate

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/cmp"
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

	for i, app := range old.Properties.AgentPoolProfiles {
		if app.Role != api.AgentPoolProfileRoleInfra {
			continue
		}

		for _, newApp := range cs.Properties.AgentPoolProfiles {
			if newApp.Role == app.Role && newApp.Count == 3 {
				old.Properties.AgentPoolProfiles[i].Count = newApp.Count
			}
		}
	}

	// validating ProvisioningState and ClusterVersion is the RP's responsibility
	old.Properties.ProvisioningState = cs.Properties.ProvisioningState
	old.Properties.ClusterVersion = cs.Properties.ClusterVersion
	old.Properties.RefreshCluster = cs.Properties.RefreshCluster

	old.Config.ComponentLogLevel = cs.Config.ComponentLogLevel

	old.Config.ImageOffer = cs.Config.ImageOffer
	old.Config.ImagePublisher = cs.Config.ImagePublisher
	old.Config.ImageSKU = cs.Config.ImageSKU
	old.Config.ImageVersion = cs.Config.ImageVersion

	old.Config.Images = cs.Config.Images
	old.Config.SecurityPatchPackages = cs.Config.SecurityPatchPackages
	old.Config.SSHSourceAddressPrefixes = cs.Config.SSHSourceAddressPrefixes

	if !reflect.DeepEqual(cs, old) {
		errs = append(errs, fmt.Errorf("invalid change %s", cmp.Diff(cs, old)))
	}

	return
}
