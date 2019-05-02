package validate

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestAPIValidateUpdate(t *testing.T) {
	tests := map[string]struct {
		f            func(*api.OpenShiftManagedCluster)
		expectedErrs []error
		externalOnly bool
	}{
		"no-op": {},
		"change compute count": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.AgentPoolProfiles[2].Count++
			},
		},
		"change compute VMSize": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.AgentPoolProfiles[2].VMSize = api.StandardF16sV2
			},
		},
		"invalid change": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Name = "new"
			},
			expectedErrs: []error{
				errors.New(`invalid change [Name: new != openshift]`),
			},
		},
		"secrets hidden": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.AuthProfile.IdentityProviders[0].Provider.(*api.AADIdentityProvider).Secret = "new"
			},
			expectedErrs: []error{
				errors.New(`invalid change [Properties.AuthProfile.IdentityProviders.slice[0].Provider.Secret: <hidden 1> != <hidden 2>]`),
			},
		},
		"provisioningstate is mutable": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.ProvisioningState = api.Creating // the RP is responsible for checking this
			},
		},
	}

	for name, test := range tests {
		var oldCs *api.OpenShiftManagedCluster
		err := yaml.Unmarshal(testOpenShiftClusterYAML, &oldCs)
		if err != nil {
			t.Fatal(err)
		}
		cs := oldCs.DeepCopy()

		if test.f != nil {
			test.f(cs)
		}
		var v APIValidator
		errs := v.Validate(cs, oldCs, false)
		if !reflect.DeepEqual(errs, test.expectedErrs) {
			t.Logf("test case %q", name)
			t.Errorf("expected errors:")
			for _, err := range test.expectedErrs {
				t.Errorf("\t%v", err)
			}
			t.Error("received errors:")
			for _, err := range errs {
				t.Errorf("\t%v", err)
			}
		}
	}
}
