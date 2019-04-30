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
		oldf         func(*api.OpenShiftManagedCluster)
		f            func(*api.OpenShiftManagedCluster)
		expectedErrs []error
	}{
		"no-op": {},
		"change log level": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Config.ComponentLogLevel.APIServer = to.IntPtr(1)
				oc.Config.ComponentLogLevel.ControllerManager = to.IntPtr(1)
				oc.Config.ComponentLogLevel.Node = to.IntPtr(1)
			},
		},
		"provisioningstate and clusterversion are mutable": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.ProvisioningState = api.Creating // the RP is responsible for checking this
				oc.Properties.ClusterVersion = "foo"           // the RP is responsible for checking this
			},
		},
		"pluginversion is immutable": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Config.PluginVersion = "latest" // the RP does this, but after validation: the user can't do this
			},
			expectedErrs: []error{
				errors.New(`invalid change [Config.PluginVersion: latest != ]`),
			},
		},
		"permitted infra scale up": {
			oldf: func(oc *api.OpenShiftManagedCluster) {
				for i, app := range oc.Properties.AgentPoolProfiles {
					if app.Role == api.AgentPoolProfileRoleInfra {
						oc.Properties.AgentPoolProfiles[i].Count = 2
					}
				}
			},
		},
		"invalid infra scale down": {
			f: func(oc *api.OpenShiftManagedCluster) {
				for i, app := range oc.Properties.AgentPoolProfiles {
					if app.Role == api.AgentPoolProfileRoleInfra {
						oc.Properties.AgentPoolProfiles[i].Count = 2
					}
				}
			},
			expectedErrs: []error{
				errors.New(`invalid change [Properties.AgentPoolProfiles.slice[1].Count: 2 != 3]`),
			},
		},
		"invalid compute scale": {
			f: func(oc *api.OpenShiftManagedCluster) {
				for i, app := range oc.Properties.AgentPoolProfiles {
					if app.Role == api.AgentPoolProfileRoleCompute {
						oc.Properties.AgentPoolProfiles[i].Count = 4
					}
				}
			},
			expectedErrs: []error{
				errors.New(`invalid change [Properties.AgentPoolProfiles.slice[2].Count: 4 != 1]`),
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
		cs := oldCs.DeepCopy()

		if test.oldf != nil {
			test.oldf(oldCs)
		}
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
