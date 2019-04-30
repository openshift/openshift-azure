package validate

import (
	"fmt"
	"reflect"

	"github.com/go-test/deep"

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
		errs = append(errs, v.validateUpdateContainerService(cs, oldCs)...)
	}

	// this limits use of RunningUnderTest variable inside our validators
	// TODO: When removed this should be part of common validators
	for _, app := range cs.Properties.AgentPoolProfiles {
		errs = append(errs, validateVMSize(&app, v.runningUnderTest)...)
	}

	return
}

func (v *APIValidator) validateUpdateContainerService(cs, oldCs *api.OpenShiftManagedCluster) (errs []error) {
	if cs == nil || oldCs == nil {
		errs = append(errs, fmt.Errorf("cs and oldCs cannot be nil"))
		return
	}

	old := oldCs.DeepCopy()

	// validating ProvisioningState is the RP's responsibility
	old.Properties.ProvisioningState = cs.Properties.ProvisioningState

	for i, app := range old.Properties.AgentPoolProfiles {
		if app.Role != api.AgentPoolProfileRoleCompute {
			continue
		}

		for _, newApp := range cs.Properties.AgentPoolProfiles {
			if newApp.Name == app.Name {
				old.Properties.AgentPoolProfiles[i].Count = newApp.Count
			}
		}
	}

	if !reflect.DeepEqual(cs, old) {
		// TODO: this is a hack because we're using deep.Equal.  To fix properly
		// we'd probably need to implement our own deep.Equal.
		csCopy := cs.DeepCopy()
		if csCopy.Properties.AuthProfile.IdentityProviders[0].Provider.(*api.AADIdentityProvider).Secret !=
			old.Properties.AuthProfile.IdentityProviders[0].Provider.(*api.AADIdentityProvider).Secret {
			csCopy.Properties.AuthProfile.IdentityProviders[0].Provider.(*api.AADIdentityProvider).Secret = "<hidden 1>"
			old.Properties.AuthProfile.IdentityProviders[0].Provider.(*api.AADIdentityProvider).Secret = "<hidden 2>"
		}
		errs = append(errs, fmt.Errorf("invalid change %s", deep.Equal(csCopy, old)))
	}

	return
}
