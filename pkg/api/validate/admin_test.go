package validate

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestAdminAPIValidate(t *testing.T) {
	tests := map[string]struct {
		f            func(*api.OpenShiftManagedCluster)
		externalOnly bool
		expectedErrs []error
	}{
		"admin api cluster create": {
			expectedErrs: []error{errors.New(`admin requests cannot create clusters`)},
		},
	}

	for name, test := range tests {
		var cs *api.OpenShiftManagedCluster
		err := yaml.Unmarshal(testOpenShiftClusterYAML, &cs)
		if err != nil {
			t.Fatal(err)
		}

		if test.f != nil {
			test.f(cs)
		}
		v := AdminAPIValidator{}
		errs := v.Validate(cs, nil)
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
