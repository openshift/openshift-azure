package config

import (
	"reflect"
	"testing"

	"github.com/ghodss/yaml"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
)

var testOpenShiftClusterYAML = []byte(`---
id: openshift
location: eastus
name: test-cluster
config:
  Version: 310
properties:
  fqdn: "console-internal.example.com"
  orchestratorProfile:
    openshiftConfig:
      PublicHostname: ""
      RouterProfiles:
      - FQDN: router-internal.example.com
        Name: router
        PublicSubdomain: ""
`)

func TestSelectDNSNames(t *testing.T) {

	tests := map[string]struct {
		f        func(*acsapi.OpenShiftManagedCluster)
		expected func(*acsapi.OpenShiftManagedCluster)
	}{

		"test no PublicHostname": {
			f: func(cs *acsapi.OpenShiftManagedCluster) {},
			expected: func(cs *acsapi.OpenShiftManagedCluster) {
				cs.Properties.OrchestratorProfile.OpenShiftConfig.PublicHostname = "console-internal.example.com"
				cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].PublicSubdomain = "router-internal.example.com"
				cs.Config.RouterLBCNamePrefix = "router-internal"
				cs.Config.MasterLBCNamePrefix = "console-internal"
			},
		},
		"test no PublicHostname for router": {
			f: func(cs *acsapi.OpenShiftManagedCluster) {
				cs.Properties.OrchestratorProfile.OpenShiftConfig.PublicHostname = "console.example.com"
			},
			expected: func(cs *acsapi.OpenShiftManagedCluster) {
				cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].PublicSubdomain = "router-internal.example.com"
				cs.Properties.OrchestratorProfile.OpenShiftConfig.PublicHostname = "console.example.com"
				cs.Config.MasterLBCNamePrefix = "console-internal"
				cs.Config.RouterLBCNamePrefix = "router-internal"
			},
		},
		"test master & router prefix configuration": {
			f: func(cs *acsapi.OpenShiftManagedCluster) {
				cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].FQDN = "router-custom.test.com"
				cs.Properties.FQDN = "master-custom.test.com"
			},
			expected: func(cs *acsapi.OpenShiftManagedCluster) {
				cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].FQDN = "router-custom.test.com"
				cs.Properties.FQDN = "master-custom.test.com"
				cs.Config.MasterLBCNamePrefix = "master-custom"
				cs.Config.RouterLBCNamePrefix = "router-custom"
				cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles[0].PublicSubdomain = "router-custom.test.com"
				cs.Properties.OrchestratorProfile.OpenShiftConfig.PublicHostname = "master-custom.test.com"
			},
		},
	}

	for name, test := range tests {
		input := new(acsapi.OpenShiftManagedCluster)
		output := new(acsapi.OpenShiftManagedCluster)
		err := yaml.Unmarshal(testOpenShiftClusterYAML, &input)
		if err != nil {
			t.Fatal(err)
		}
		// TODO: This can be replaced by deepCopy of input before we populate it
		err = yaml.Unmarshal(testOpenShiftClusterYAML, &output)
		if err != nil {
			t.Fatal(err)
		}

		if test.f != nil {
			test.f(input)
		}
		if test.expected != nil {
			test.expected(output)
		}

		selectDNSNames(input)

		if !reflect.DeepEqual(input, output) {
			t.Errorf("%v: SelectDNSNames test returned unexpected result \n %#v != %#v", name, input, output)
		}

	}
}
