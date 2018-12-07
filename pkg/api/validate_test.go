package api

import (
	"errors"
	"net"
	"reflect"
	"testing"

	"github.com/ghodss/yaml"

	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
)

var testOpenShiftClusterYAML = []byte(`---
location: eastus
name: openshift
properties:
  openShiftVersion: v3.11
  fqdn: example.eastus.cloudapp.azure.com
  authProfile:
    identityProviders:
    - name: Azure AD
      provider:
        kind: AADIdentityProvider
        clientId: aadClientId
        secret: aadClientSecret
        tenantId: aadTenantId
  routerProfiles:
  - name: default
    publicSubdomain: test.example.com
    fqdn: router-fqdn.eastus.cloudapp.azure.com
  networkProfile:
    vnetCidr: 10.0.0.0/8
  masterPoolProfile:
    count: 3
    vmSize: Standard_D2s_v3
    subnetCidr: 10.0.0.0/24
  agentPoolProfiles:
  - name: infra
    role: infra
    count: 2
    vmSize: Standard_D2s_v3
    osType: Linux
    subnetCidr: 10.0.0.0/24
  - name: myCompute
    role: compute
    count: 1
    vmSize: Standard_D2s_v3
    osType: Linux
    subnetCidr: 10.0.0.0/24
`)

func TestValidate(t *testing.T) {
	tests := map[string]struct {
		f            func(*OpenShiftManagedCluster)
		expectedErrs []error
		externalOnly bool
		simulateProd bool // this defaults to false, that way I don't have to define it everywhere
	}{
		"test yaml parsing": { // test yaml parsing

		},
		"simulating prod, Standard_D2s_v3": {
			f: func(oc *OpenShiftManagedCluster) {
				for i := range oc.Properties.AgentPoolProfiles {
					oc.Properties.AgentPoolProfiles[i].VMSize = "Standard_D2s_v3"
				}
			},
			simulateProd: true,
			expectedErrs: []error{
				errors.New(`invalid properties.masterPoolProfile.vmSize "Standard_D2s_v3"`),
				errors.New(`invalid properties.agentPoolProfiles["infra"].vmSize "Standard_D2s_v3"`),
				errors.New(`invalid properties.agentPoolProfiles["myCompute"].vmSize "Standard_D2s_v3"`),
			},
		},
		"simulating prod, Standard_D8s_v3": {
			f: func(oc *OpenShiftManagedCluster) {
				for i := range oc.Properties.AgentPoolProfiles {
					oc.Properties.AgentPoolProfiles[i].VMSize = "Standard_D8s_v3"
				}
			},
			simulateProd: true,
		},
		"running under test, Standard_D8s_v3": {
			f: func(oc *OpenShiftManagedCluster) {
				for i := range oc.Properties.AgentPoolProfiles {
					oc.Properties.AgentPoolProfiles[i].VMSize = "Standard_D8s_v3"
				}
			},
		},
		"running under test, Standard_D2s_v3": {
			f: func(oc *OpenShiftManagedCluster) {
				for i := range oc.Properties.AgentPoolProfiles {
					oc.Properties.AgentPoolProfiles[i].VMSize = "Standard_D2s_v3"
				}
			},
		},
		"empty location": {
			f: func(oc *OpenShiftManagedCluster) { oc.Location = "" },
			expectedErrs: []error{
				errors.New(`invalid location ""`),
				errors.New(`invalid properties.routerProfiles["default"].fqdn "router-fqdn.eastus.cloudapp.azure.com"`),
				errors.New(`invalid properties.fqdn "example.eastus.cloudapp.azure.com"`),
			},
		},
		"name": {
			f:            func(oc *OpenShiftManagedCluster) { oc.Name = "" },
			expectedErrs: []error{errors.New(`invalid name ""`)},
		},
		"openshift config invalid api fqdn": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.FQDN = ""
			},
			expectedErrs: []error{errors.New(`invalid properties.fqdn ""`)},
		},
		"test external only false - invalid fqdn fails": {
			f:            func(oc *OpenShiftManagedCluster) { oc.Properties.FQDN = "()" },
			expectedErrs: []error{errors.New(`invalid properties.fqdn "()"`)},
			externalOnly: false,
		},
		"provisioning state bad": {
			f:            func(oc *OpenShiftManagedCluster) { oc.Properties.ProvisioningState = "bad" },
			expectedErrs: []error{errors.New(`invalid properties.provisioningState "bad"`)},
		},
		"provisioning state Creating": {
			f: func(oc *OpenShiftManagedCluster) { oc.Properties.ProvisioningState = "Creating" },
		},
		"provisioning state Failed": {
			f: func(oc *OpenShiftManagedCluster) { oc.Properties.ProvisioningState = "Failed" },
		},
		"provisioning state Updating": {
			f: func(oc *OpenShiftManagedCluster) { oc.Properties.ProvisioningState = "Updating" },
		},
		"provisioning state Succeeded": {
			f: func(oc *OpenShiftManagedCluster) { oc.Properties.ProvisioningState = "Succeeded" },
		},
		"provisioning state Deleting": {
			f: func(oc *OpenShiftManagedCluster) { oc.Properties.ProvisioningState = "Deleting" },
		},
		"provisioning state Migrating": {
			f: func(oc *OpenShiftManagedCluster) { oc.Properties.ProvisioningState = "Migrating" },
		},
		"provisioning state Upgrading": {
			f: func(oc *OpenShiftManagedCluster) { oc.Properties.ProvisioningState = "Upgrading" },
		},
		"provisioning state empty": {
			f: func(oc *OpenShiftManagedCluster) { oc.Properties.ProvisioningState = "" },
		},
		"openshift version good": {
			f: func(oc *OpenShiftManagedCluster) { oc.Properties.OpenShiftVersion = "v3.11" },
		},
		"openshift version bad": {
			f:            func(oc *OpenShiftManagedCluster) { oc.Properties.OpenShiftVersion = "" },
			expectedErrs: []error{errors.New(`invalid properties.openShiftVersion ""`)},
		},
		"openshift config empty public hostname": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.PublicHostname = ""
			},
		},
		"openshift config invalid public hostname": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.PublicHostname = "www.example.com"
			},
			expectedErrs: []error{errors.New(`invalid properties.publicHostname "www.example.com"`)},
		},
		"network profile bad vnetCidr": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.NetworkProfile.VnetCIDR = "foo"
			},
			expectedErrs: []error{errors.New(`invalid properties.networkProfile.vnetCidr "foo"`)},
		},
		"network profile invalid vnetCidr": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.NetworkProfile.VnetCIDR = "192.168.0.0/16"
			},
			expectedErrs: []error{
				errors.New(`invalid properties.agentPoolProfiles["master"].subnetCidr "10.0.0.0/24": not contained in properties.networkProfile.vnetCidr "192.168.0.0/16"`),
				errors.New(`invalid properties.agentPoolProfiles["infra"].subnetCidr "10.0.0.0/24": not contained in properties.networkProfile.vnetCidr "192.168.0.0/16"`),
				errors.New(`invalid properties.agentPoolProfiles["myCompute"].subnetCidr "10.0.0.0/24": not contained in properties.networkProfile.vnetCidr "192.168.0.0/16"`),
			},
		},
		"network profile valid peerVnetId": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.NetworkProfile.PeerVnetID = "/subscriptions/b07e8fae-2f3f-4769-8fa8-8570b426ba13/resourceGroups/test/providers/Microsoft.Network/virtualNetworks/vnet"
			},
		},
		"network profile invalid peerVnetId": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.NetworkProfile.PeerVnetID = "foo"
			},
			expectedErrs: []error{errors.New(`invalid properties.networkProfile.peerVnetId "foo"`)},
		},
		"router profile duplicate names": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.RouterProfiles =
					append(oc.Properties.RouterProfiles,
						oc.Properties.RouterProfiles[0])
			},
			expectedErrs: []error{errors.New(`duplicate properties.routerProfiles "default"`)},
		},
		"router profile invalid name": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.RouterProfiles[0].Name = "foo"
			},
			// two errors expected here because we require the default profile
			expectedErrs: []error{errors.New(`invalid properties.routerProfiles["foo"]`),
				errors.New(`invalid properties.routerProfiles["default"]`)},
		},
		"router profile empty name": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.RouterProfiles[0].Name = ""
			},
			// same as above with 2 errors but additional validate on the individual profile yeilds a third
			// this is not very user friendly but testing as is for now
			// TODO fix
			expectedErrs: []error{errors.New(`invalid properties.routerProfiles[""]`),
				errors.New(`invalid properties.routerProfiles[""].name ""`),
				errors.New(`invalid properties.routerProfiles["default"]`)},
		},
		"router empty public subdomain": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.RouterProfiles[0].PublicSubdomain = ""
			},
		},
		"router invalid public subdomain": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.RouterProfiles[0].PublicSubdomain = "()"
			},
			expectedErrs: []error{errors.New(`invalid properties.routerProfiles["default"].publicSubdomain "()"`)},
		},
		"test external only true - unset router profile does not fail": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.RouterProfiles = nil
			},
			externalOnly: true,
		},
		"test external only false - unset router profile does fail": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.RouterProfiles = nil
			},
			expectedErrs: []error{errors.New(`invalid properties.routerProfiles["default"]`)},
			externalOnly: false,
		},
		"test external only false - invalid router profile does fail": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.RouterProfiles[0].FQDN = "()"
			},
			expectedErrs: []error{errors.New(`invalid properties.routerProfiles["default"].fqdn "()"`)},
			externalOnly: false,
		},
		"agent pool profile duplicate name": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.AgentPoolProfiles = append(
					oc.Properties.AgentPoolProfiles,
					oc.Properties.AgentPoolProfiles[1])
			},
			expectedErrs: []error{errors.New(`duplicate role "infra" in properties.agentPoolProfiles["infra"]`)},
		},
		"agent pool profile invalid infra name": {
			f: func(oc *OpenShiftManagedCluster) {
				for i, app := range oc.Properties.AgentPoolProfiles {
					if app.Role == AgentPoolProfileRoleInfra {
						oc.Properties.AgentPoolProfiles[i].Name = "foo"
					}
				}
			},
			expectedErrs: []error{
				errors.New(`invalid properties.agentPoolProfiles["foo"].name "foo"`),
			},
		},
		"agent pool profile invalid compute name": {
			f: func(oc *OpenShiftManagedCluster) {
				for i, app := range oc.Properties.AgentPoolProfiles {
					if app.Role == AgentPoolProfileRoleCompute {
						oc.Properties.AgentPoolProfiles[i].Name = "$"
					}
				}
			},
			expectedErrs: []error{
				errors.New(`invalid properties.agentPoolProfiles["$"].name "$"`),
			},
		},
		"agent pool profile invalid vm size": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.AgentPoolProfiles[1].VMSize = VMSize("SuperBigVM")
			},
			expectedErrs: []error{
				errors.New(`invalid properties.agentPoolProfiles["infra"].vmSize "SuperBigVM"`),
				errors.New(`invalid properties.agentPoolProfiles.vmSize "SuperBigVM": master and infra vmSizes must match`),
			},
		},
		"agent pool unmatched subnet cidr": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.AgentPoolProfiles[2].SubnetCIDR = "10.0.1.0/24"
			},
			expectedErrs: []error{errors.New(`invalid properties.agentPoolProfiles.subnetCidr "10.0.1.0/24": all subnetCidrs must match`)},
		},
		"agent pool bad subnet cidr": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.AgentPoolProfiles[2].SubnetCIDR = "foo"
			},
			expectedErrs: []error{
				errors.New(`invalid properties.agentPoolProfiles.subnetCidr "foo": all subnetCidrs must match`),
				errors.New(`invalid properties.agentPoolProfiles["myCompute"].subnetCidr "foo"`),
			},
		},
		"agent pool subnet cidr clash cluster": {
			f: func(oc *OpenShiftManagedCluster) {
				for i := range oc.Properties.AgentPoolProfiles {
					oc.Properties.AgentPoolProfiles[i].SubnetCIDR = "10.128.0.0/24"
				}
			},
			expectedErrs: []error{
				errors.New(`invalid properties.agentPoolProfiles["master"].subnetCidr "10.128.0.0/24": overlaps with cluster network "10.128.0.0/14"`),
				errors.New(`invalid properties.agentPoolProfiles["infra"].subnetCidr "10.128.0.0/24": overlaps with cluster network "10.128.0.0/14"`),
				errors.New(`invalid properties.agentPoolProfiles["myCompute"].subnetCidr "10.128.0.0/24": overlaps with cluster network "10.128.0.0/14"`),
			},
		},
		"agent pool subnet cidr clash service": {
			f: func(oc *OpenShiftManagedCluster) {
				oc.Properties.NetworkProfile.VnetCIDR = "172.0.0.0/8"
				for i := range oc.Properties.AgentPoolProfiles {
					oc.Properties.AgentPoolProfiles[i].SubnetCIDR = "172.30.0.0/16"
				}
			},
			expectedErrs: []error{
				errors.New(`invalid properties.agentPoolProfiles["master"].subnetCidr "172.30.0.0/16": overlaps with service network "172.30.0.0/16"`),
				errors.New(`invalid properties.agentPoolProfiles["infra"].subnetCidr "172.30.0.0/16": overlaps with service network "172.30.0.0/16"`),
				errors.New(`invalid properties.agentPoolProfiles["myCompute"].subnetCidr "172.30.0.0/16": overlaps with service network "172.30.0.0/16"`),
			},
		},
		"agent pool bad master count": {
			f: func(oc *OpenShiftManagedCluster) {
				for i, app := range oc.Properties.AgentPoolProfiles {
					if app.Role == AgentPoolProfileRoleMaster {
						oc.Properties.AgentPoolProfiles[i].Count = 1
					}
				}
			},
			expectedErrs: []error{errors.New(`invalid properties.masterPoolProfile.count 1`)},
		},
		//we dont check authProfile because it is non pointer struct. Which is all zero values.
		"authProfile.identityProviders empty": {
			f:            func(oc *OpenShiftManagedCluster) { oc.Properties.AuthProfile = AuthProfile{} },
			expectedErrs: []error{errors.New(`invalid properties.authProfile.identityProviders length`)},
		},
		"AADIdentityProvider secret empty": {
			f: func(oc *OpenShiftManagedCluster) {
				aadIdentityProvider := &AADIdentityProvider{
					Kind:     "AADIdentityProvider",
					ClientID: "clientId",
					Secret:   "",
					TenantID: "tenantId",
				}
				oc.Properties.AuthProfile.IdentityProviders[0].Provider = aadIdentityProvider
				oc.Properties.AuthProfile.IdentityProviders[0].Name = "Azure AD"
			},
			expectedErrs: []error{errors.New(`invalid properties.authProfile.AADIdentityProvider clientId ""`)},
		},
		"AADIdentityProvider clientId empty": {
			f: func(oc *OpenShiftManagedCluster) {
				aadIdentityProvider := &AADIdentityProvider{
					Kind:     "AADIdentityProvider",
					ClientID: "",
					Secret:   "aadClientSecret",
					TenantID: "tenantId",
				}
				oc.Properties.AuthProfile.IdentityProviders[0].Provider = aadIdentityProvider
				oc.Properties.AuthProfile.IdentityProviders[0].Name = "Azure AD"
			},
			expectedErrs: []error{errors.New(`invalid properties.authProfile.AADIdentityProvider clientId ""`)},
		},
		"AADIdentityProvider tenantId empty": {
			f: func(oc *OpenShiftManagedCluster) {
				aadIdentityProvider := &AADIdentityProvider{
					Kind:     "AADIdentityProvider",
					ClientID: "test",
					Secret:   "aadClientSecret",
					TenantID: "",
				}
				oc.Properties.AuthProfile.IdentityProviders[0].Provider = aadIdentityProvider
				oc.Properties.AuthProfile.IdentityProviders[0].Name = "Azure AD"
			},
			expectedErrs: []error{errors.New(`invalid properties.authProfile.AADIdentityProvider tenantId ""`)},
		},
	}

	for name, test := range tests {
		var oc *v20180930preview.OpenShiftManagedCluster
		err := yaml.Unmarshal(testOpenShiftClusterYAML, &oc)
		if err != nil {
			t.Fatal(err)
		}

		// TODO we're hoping conversion is correct. Change this to a known valid config
		cs, err := ConvertFromV20180930preview(oc, nil)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", name, err)
		}
		if test.f != nil {
			test.f(cs)
		}
		v := Validator{runningUnderTest: !test.simulateProd}
		errs := v.Validate(cs, nil, test.externalOnly)
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

func TestIsValidCloudAppHostname(t *testing.T) {
	invalidFqdns := []string{
		"invalid.random.domain",
		"too.long.domain.cloudapp.azure.com",
		"invalid#characters#domain.westus2.cloudapp.azure.com",
		"wronglocation.eastus.cloudapp.azure.com",
		"123.eastus.cloudapp.azure.com",
		"-abc.eastus.cloudapp.azure.com",
		"abcdefghijklmnopqrstuvwxzyabcdefghijklmnopqrstuvwxzyabcdefghijkl.eastus.cloudapp.azure.com",
		"a/b/c.eastus.cloudapp.azure.com",
		".eastus.cloudapp.azure.com",
		"Thisisatest.eastus.cloudapp.azure.com",
	}
	for _, invalidFqdn := range invalidFqdns {
		if isValidCloudAppHostname(invalidFqdn, "westus2") {
			t.Errorf("invalid FQDN passed test: %s", invalidFqdn)
		}
	}
	validFqdns := []string{
		"example.westus2.cloudapp.azure.com",
		"test-dashes.westus2.cloudapp.azure.com",
		"test123.westus2.cloudapp.azure.com",
		"test-123.westus2.cloudapp.azure.com",
	}
	for _, validFqdn := range validFqdns {
		if !isValidCloudAppHostname(validFqdn, "westus2") {
			t.Errorf("Valid FQDN failed to pass test: %s", validFqdn)
		}
	}
}

func TestIsValidIPV4CIDR(t *testing.T) {
	for _, test := range []struct {
		cidr  string
		valid bool
	}{
		{
			cidr: "",
		},
		{
			cidr: "foo",
		},
		{
			cidr: "::/0",
		},
		{
			cidr: "192.168.0.1/24",
		},
		{
			cidr:  "192.168.0.0/24",
			valid: true,
		},
	} {
		valid := isValidIPV4CIDR(test.cidr)
		if valid != test.valid {
			t.Errorf("%s: unexpected result %v", test.cidr, valid)
		}
	}
}

func TestVnetContainsSubnet(t *testing.T) {
	for i, test := range []struct {
		vnetCidr   string
		subnetCidr string
		valid      bool
	}{
		{
			vnetCidr:   "10.0.0.0/16",
			subnetCidr: "192.168.0.0/16",
		},
		{
			vnetCidr:   "10.0.0.0/16",
			subnetCidr: "10.0.0.0/8",
		},
		{
			vnetCidr:   "10.0.0.0/16",
			subnetCidr: "10.0.128.0/15",
		},
		{
			vnetCidr:   "10.0.0.0/8",
			subnetCidr: "10.0.0.0/16",
			valid:      true,
		},
		{
			vnetCidr:   "10.0.0.0/8",
			subnetCidr: "10.0.0.0/8",
			valid:      true,
		},
	} {
		_, vnet, err := net.ParseCIDR(test.vnetCidr)
		if err != nil {
			t.Fatal(err)
		}

		_, subnet, err := net.ParseCIDR(test.subnetCidr)
		if err != nil {
			t.Fatal(err)
		}

		valid := vnetContainsSubnet(vnet, subnet)
		if valid != test.valid {
			t.Errorf("%d: unexpected result %v", i, valid)
		}
	}
}

func TestValidateUpdateContainerService(t *testing.T) {
	var twoAgents = []byte(`---
location: eastus
name: openshift
properties:
  publicHostname: donotchange
  routerProfiles:
  - name: default
    publicSubdomain: test.example.com
  agentPoolProfiles:
  - name: infra
    role: infra
    count: 1
    vmSize: Standard_D2s_v3
    osType: Linux
  - name: myCompute
    role: compute
    count: 1
    vmSize: Standard_D2s_v3
    osType: Linux
`)

	tests := map[string]struct {
		newAgentCount    int
		oldAgentCount    int
		newHostNameValue string
		wantErrs         []error
	}{
		"good-2": {
			newAgentCount: 1,
			oldAgentCount: 1,
		},
		"good-2-count-not-important": {
			newAgentCount: 5,
			oldAgentCount: 2,
		},
		"bad-field-change": {
			newAgentCount:    1,
			oldAgentCount:    1,
			newHostNameValue: "different",
			wantErrs:         []error{errors.New("invalid change [Properties.PublicHostname: different != donotchange]")},
		},
	}

	for name, tt := range tests {
		var newCs *OpenShiftManagedCluster
		var oldCs *OpenShiftManagedCluster

		err := yaml.Unmarshal(twoAgents, &oldCs)
		if err != nil {
			t.Fatal(err)
		}
		newCs = oldCs.DeepCopy()

		for i := range oldCs.Properties.AgentPoolProfiles {
			oldCs.Properties.AgentPoolProfiles[i].Count = tt.oldAgentCount
		}
		for i := range newCs.Properties.AgentPoolProfiles {
			newCs.Properties.AgentPoolProfiles[i].Count = tt.newAgentCount
		}
		if tt.newHostNameValue != "" {
			newCs.Properties.PublicHostname = tt.newHostNameValue
		}

		var v Validator
		gotErrs := v.validateUpdateContainerService(newCs, oldCs, false)
		if !reflect.DeepEqual(gotErrs, tt.wantErrs) {
			t.Errorf("validateUpdateContainerService:%s() = %v, want %v", name, gotErrs, tt.wantErrs)
		}
	}
}
