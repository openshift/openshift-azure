package validate

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestAdminAPIValidate(t *testing.T) {
	var v AdminAPIValidator
	errs := v.Validate(&api.OpenShiftManagedCluster{}, nil)
	if len(errs) != 1 || errs[0].Error() != `admin requests cannot create clusters` {
		t.Errorf("unexpected validate output %#v", errs)
	}
}

func TestAdminAPIValidateUpdate(t *testing.T) {
	tests := map[string]struct {
		pluginVersion string
		f             func(*api.OpenShiftManagedCluster)
		expectedErrs  []error
	}{
		"no-op": {},
		"change log level": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Config.ComponentLogLevel.APIServer = to.IntPtr(1)
				oc.Config.ComponentLogLevel.ControllerManager = to.IntPtr(1)
				oc.Config.ComponentLogLevel.Node = to.IntPtr(1)
			},
		},
		"invalid upgrade": {
			pluginVersion: "v2.0",
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Config.PluginVersion = "latest"
			},
			expectedErrs: []error{
				errors.New(`invalid change [Config.PluginVersion: latest != v2.0]`),
			},
		},
		"permitted upgrade": {
			pluginVersion: "v3.0",
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Config.PluginVersion = "latest"
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
	}

	for name, test := range tests {
		var oldCs *api.OpenShiftManagedCluster
		err := yaml.Unmarshal(testOpenShiftClusterYAML, &oldCs)
		if err != nil {
			t.Fatal(err)
		}
		oldCs.Config.PluginVersion = test.pluginVersion
		cs := oldCs.DeepCopy()

		if test.f != nil {
			test.f(cs)
		}
		var v AdminAPIValidator
		errs := v.Validate(cs, oldCs)
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
