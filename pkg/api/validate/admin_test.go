package validate

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
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
		var oc *v20180930preview.OpenShiftManagedCluster
		err := yaml.Unmarshal(testOpenShiftClusterYAML, &oc)
		if err != nil {
			t.Fatal(err)
		}

		// TODO we're hoping conversion is correct. Change this to a known valid config
		cs, err := api.ConvertFromV20180930preview(oc, nil)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", name, err)
		}
		if test.f != nil {
			test.f(cs)
		}
		v := AdminAPIValidator{}
		errs := v.Validate(cs, nil, false)
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
