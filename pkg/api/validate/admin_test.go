package validate

import (
	"regexp"
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
		expectedErrs []*regexp.Regexp
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
		"pluginversion is immutable": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Config.PluginVersion = "latest" // the RP does this, but after validation: the user can't do this
			},
			expectedErrs: []*regexp.Regexp{
				regexp.MustCompile(`PluginVersion:\s+?"latest"(?s).+?PluginVersion:\s+?""`),
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
			expectedErrs: []*regexp.Regexp{
				regexp.MustCompile(`invalid properties.agentPoolProfiles\["infra"\].count 2`),
				regexp.MustCompile(`Count:\s+?2(?s).+?Count:\s+?3`),
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
			expectedErrs: []*regexp.Regexp{
				regexp.MustCompile(`Count:\s+?4(?s).+?Count:\s+?1`),
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
