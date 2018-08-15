package validate

import (
	"errors"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/api/v1"
)

/*
name: openshift
location: eastus
properties:
  openShiftVersion: "$DEPLOY_VERSION"
  publicHostname: openshift.$RESOURCEGROUP.$DNS_DOMAIN
  routerProfiles:
  - name: default
    publicSubdomain: $RESOURCEGROUP.$DNS_DOMAIN
  agentPoolProfiles:
  - name: master
    role: master
    count: 3
    vmSize: Standard_D2s_v3
    osType: Linux
  - name: infra
    role: infra
    count: 1
    vmSize: Standard_D2s_v3
    osType: Linux
  - name: compute
    role: compute
    count: 1
    vmSize: Standard_D2s_v3
    osType: Linux
  servicePrincipalProfile:
    clientID: $AZURE_CLIENT_ID
    secret: $AZURE_CLIENT_SECRET
*/

var testOpenShiftClusterYAML = []byte(`---
location: eastus
name: openshift
properties:
  openShiftVersion: v3.10
  publicHostname: openshift.test.example.com
  fqdn: "www.example.com"
  routerProfiles:
  - name: default
    publicSubdomain: test.example.com
  masterPoolProfile:
    name: master
    count: 3
    vmSize: Standard_D2s_v3
    osType: Linux
  agentPoolProfiles: 
  - name: infra
    role: infra
    count: 1
    vmSize: Standard_D2s_v3
    osType: Linux
  - name: compute
    role: compute
    count: 1
    vmSize: Standard_D2s_v3
    osType: Linux
  servicePrincipalProfile:
    clientID: client_id
    secret: client_secret
`)

func TestValidate(t *testing.T) {

	tests := map[string]struct {
		f            func(*v1.OpenShiftManagedCluster)
		expectedErrs []error
		externalOnly bool
	}{
		"test yaml parsing": { // test yaml parsing

		},
		"test version": {
			f:            func(oc *v1.OpenShiftManagedCluster) { oc.Properties.OpenShiftVersion = "v3.11" },
			expectedErrs: []error{errors.New("invalid properties.openShiftVersion \"v3.11\"")},
		},
		"test Location": {
			f:            func(oc *v1.OpenShiftManagedCluster) { oc.Location = "" },
			expectedErrs: []error{errors.New("invalid location \"\"")},
		},
		"test Name": {
			f:            func(oc *v1.OpenShiftManagedCluster) { oc.Name = "" },
			expectedErrs: []error{errors.New("invalid name \"\"")},
		},
		"test ProvisioningState": {
			f:            func(oc *v1.OpenShiftManagedCluster) { oc.Properties.ProvisioningState = "testing" },
			expectedErrs: []error{errors.New("invalid properties.provisioningState \"testing\"")},
		},
		"test master count": {
			f:            func(oc *v1.OpenShiftManagedCluster) { oc.Properties.MasterPoolProfile.Count = 1 },
			expectedErrs: []error{errors.New("invalid masterPoolProfile.count 1")},
		},
		"test external only true - unset fqdn does not fail": {
			f:            func(oc *v1.OpenShiftManagedCluster) { oc.Properties.FQDN = "" },
			externalOnly: true,
		},
		"test external only false - unset fqdn fails": {
			f:            func(oc *v1.OpenShiftManagedCluster) { oc.Properties.FQDN = "" },
			expectedErrs: []error{errors.New("invalid properties.fqdn \"\"")},
			externalOnly: false,
		},
		"test external only false - invalid fqdn fails": {
			f:            func(oc *v1.OpenShiftManagedCluster) { oc.Properties.FQDN = "()" },
			expectedErrs: []error{errors.New("invalid properties.fqdn \"()\"")},
			externalOnly: false,
		},
		"test external only true - unset router profile does not fail": {
			f:            func(oc *v1.OpenShiftManagedCluster) { oc.Properties.RouterProfiles = nil },
			externalOnly: true,
		},
		"test external only false - unset router profile does fail": {
			f:            func(oc *v1.OpenShiftManagedCluster) { oc.Properties.RouterProfiles = nil },
			expectedErrs: []error{errors.New("invalid properties.routerProfiles[\"default\"]")},
			externalOnly: false,
		},
		"test external only false - invalid router profile does fail": {
			f:            func(oc *v1.OpenShiftManagedCluster) { oc.Properties.RouterProfiles[0].FQDN = "()" },
			expectedErrs: []error{errors.New("invalid properties.routerProfiles[\"default\"].fqdn \"()\"")},
			externalOnly: false,
		},
	}

	for name, test := range tests {
		var oc *v1.OpenShiftManagedCluster
		err := yaml.Unmarshal(testOpenShiftClusterYAML, &oc)
		if err != nil {
			t.Fatal(err)
		}

		if test.f != nil {
			test.f(oc)
		}

		// TODO quick fix but means we're hoping conversion is correct.
		cs := api.ConvertV1OpenShiftManagedClusterToOpenShiftManagedCluster(oc)
		errs := Validate(cs, nil, test.externalOnly)
		if !reflect.DeepEqual(errs, test.expectedErrs) {
			t.Errorf("%s expected errors %#v but received %#v", name, spew.Sprint(test.expectedErrs), spew.Sprint(errs))
		}

	}
}
