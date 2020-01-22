package validate

import (
	"regexp"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestAPIValidateUpdate(t *testing.T) {
	tests := map[string]struct {
		f            func(*api.OpenShiftManagedCluster)
		expectedErrs []*regexp.Regexp
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
		"set refreshcluster": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.RefreshCluster = to.BoolPtr(true)
			},
		},
		"unset refreshcluster": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.RefreshCluster = to.BoolPtr(false)
			},
		},
		"invalid change": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Name = "new"
			},
			expectedErrs: []*regexp.Regexp{
				regexp.MustCompile(`Name:\s+?"new"(?s).+?Name:\s+?"openshift"`),
			},
		},
		"change AADIdentityProvider": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.AuthProfile.IdentityProviders[0].Provider.(*api.AADIdentityProvider).Secret = "new"
			},
		},
		"provisioningstate is mutable": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.ProvisioningState = api.Creating // the RP is responsible for checking this
			},
		},
		"config structure is not exposed if changed": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Config.EtcdMetricsPassword = "test"
			},
			expectedErrs: []*regexp.Regexp{
				// when config struct changed we should print ONLY error
				// No diff output should be printed from the struct
				regexp.MustCompile(`invalid change $`),
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
		v := &APIValidator{roleLister: &DummyRoleLister{}}
		errs := v.Validate(cs, oldCs, false)
		if len(test.expectedErrs) != len(errs) {
			t.Errorf("%s: expected %d errors, got %d", name, len(test.expectedErrs), len(errs))
		}

		for i, exp := range test.expectedErrs {
			if !exp.MatchString(errs[i].Error()) {
				t.Errorf("%s: error at index %d doesn't match \"%v\". Error: %v", name, i, exp, errs[i])
			}
		}
	}
}
