package validate

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
)

var testOpenShiftClusterYAML = []byte(`
location: eastus
name: openshift
properties:
  agentPoolProfiles:
  - count: 3
    name: master
    osType: Linux
    role: master
    subnetCidr: 10.0.0.0/24
    vmSize: Standard_D4s_v3
  - count: 3
    name: infra
    osType: Linux
    role: infra
    subnetCidr: 10.0.0.0/24
    vmSize: Standard_D4s_v3
  - count: 1
    name: mycompute
    osType: Linux
    role: compute
    subnetCidr: 10.0.0.0/24
    vmSize: Standard_D4s_v3
  apiCertProfile:
    keyVaultSecretURL: https://myvault.vault.azure.net/secrets/secret
  authProfile:
    identityProviders:
    - name: Azure AD
      provider:
        clientId: 00000000-0000-0000-0000-000000000000
        kind: AADIdentityProvider
        secret: secret
        tenantId: 00000000-0000-0000-0000-000000000000
  azProfile:
    resourceGroup: resourcegroup
    subscriptionId: 00000000-0000-0000-0000-000000000000
    tenantId: 00000000-0000-0000-0000-000000000000
  fqdn: example.eastus.cloudapp.azure.com
  masterServicePrincipalProfile:
    clientID: 00000000-0000-0000-0000-000000000000
    secret: secret
  monitorProfile:
    enabled: true
    workspaceResourceID: /subscriptions/foo/resourceGroups/bar/providers/Microsoft.OperationalInsights/workspaces/baz
    workspaceID: 00000000-0000-0000-0000-000000000000
    workspaceKey: a2V5Cg==
  networkProfile:
    vnetCidr: 10.0.0.0/8
    managementSubnetCidr: 10.0.1.0/24
  openShiftVersion: v3.11
  publicHostname: test.example.com
  routerProfiles:
  - fqdn: router-fqdn.eastus.cloudapp.azure.com
    name: default
    publicSubdomain: test.example.com
    routerCertProfile:
      keyVaultSecretURL: https://myvault.vault.azure.net/secrets/secret
  workerServicePrincipalProfile:
    clientID: 00000000-0000-0000-0000-000000000000
    secret: secret
`)

func TestValidate(t *testing.T) {
	tests := map[string]struct {
		f            func(*api.OpenShiftManagedCluster)
		expectedErrs []error
		externalOnly bool
		simulateProd bool // this defaults to false, that way I don't have to define it everywhere
	}{
		"test yaml parsing": { // test yaml parsing

		},
		"simulating prod, Standard_D2s_v3": {
			f: func(oc *api.OpenShiftManagedCluster) {
				for i := range oc.Properties.AgentPoolProfiles {
					oc.Properties.AgentPoolProfiles[i].VMSize = "Standard_D2s_v3"
				}
			},
			simulateProd: true,
			expectedErrs: []error{
				errors.New(`invalid properties.masterPoolProfile.vmSize "Standard_D2s_v3"`),
				errors.New(`invalid properties.agentPoolProfiles["infra"].vmSize "Standard_D2s_v3"`),
				errors.New(`invalid properties.agentPoolProfiles["mycompute"].vmSize "Standard_D2s_v3"`),
			},
		},
		"simulating prod, Standard_D8s_v3": {
			f: func(oc *api.OpenShiftManagedCluster) {
				for i := range oc.Properties.AgentPoolProfiles {
					oc.Properties.AgentPoolProfiles[i].VMSize = "Standard_D8s_v3"
				}
			},
			simulateProd: true,
		},
		"simulating prod, master and infra nodes don't match": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.AgentPoolProfiles[0].VMSize = "Standard_D4s_v3"
				oc.Properties.AgentPoolProfiles[1].VMSize = "Standard_D8s_v3"
				oc.Properties.AgentPoolProfiles[2].VMSize = "Standard_D8s_v3"
			},
			simulateProd: true,
			expectedErrs: []error{
				errors.New(`invalid properties.agentPoolProfiles.vmSize "Standard_D8s_v3": master and infra vmSizes must match`),
			},
		},
		"simulating prod, all sizes outside the valid types": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.AgentPoolProfiles[0].VMSize = "Standard_D64s_v3"
				oc.Properties.AgentPoolProfiles[1].VMSize = "Standard_D64s_v3"
				oc.Properties.AgentPoolProfiles[2].VMSize = "Standard_F64s_v3"
			},
			simulateProd: true,
			expectedErrs: []error{
				errors.New(`invalid properties.masterPoolProfile.vmSize "Standard_D64s_v3"`),
				errors.New(`invalid properties.agentPoolProfiles["infra"].vmSize "Standard_D64s_v3"`),
				errors.New(`invalid properties.agentPoolProfiles["mycompute"].vmSize "Standard_F64s_v3"`),
			},
		},
		"running under test, Standard_D8s_v3": {
			f: func(oc *api.OpenShiftManagedCluster) {
				for i := range oc.Properties.AgentPoolProfiles {
					oc.Properties.AgentPoolProfiles[i].VMSize = "Standard_D8s_v3"
				}
			},
		},
		"running under test, Standard_D2s_v3": {
			f: func(oc *api.OpenShiftManagedCluster) {
				for i := range oc.Properties.AgentPoolProfiles {
					oc.Properties.AgentPoolProfiles[i].VMSize = "Standard_D2s_v3"
				}
			},
		},
		"empty location": {
			f: func(oc *api.OpenShiftManagedCluster) { oc.Location = "" },
			expectedErrs: []error{
				errors.New(`invalid properties.fqdn "example.eastus.cloudapp.azure.com"`),
				errors.New(`invalid properties.routerProfiles["default"].fqdn "router-fqdn.eastus.cloudapp.azure.com"`),
				errors.New(`invalid location ""`),
			},
		},
		"invalid location": {
			f: func(oc *api.OpenShiftManagedCluster) { oc.Location = "West US 2" },
			expectedErrs: []error{
				errors.New(`invalid properties.fqdn "example.eastus.cloudapp.azure.com"`),
				errors.New(`invalid properties.routerProfiles["default"].fqdn "router-fqdn.eastus.cloudapp.azure.com"`),
				errors.New(`invalid location "West US 2"`),
			},
		},
		"empty name": {
			f:            func(oc *api.OpenShiftManagedCluster) { oc.Name = "" },
			expectedErrs: []error{errors.New(`invalid name ""`)},
		},
		"invalid name": {
			f:            func(oc *api.OpenShiftManagedCluster) { oc.Name = "cluster name" },
			expectedErrs: []error{errors.New(`invalid name "cluster name"`)},
		},
		"openshift config invalid api fqdn": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.FQDN = ""
			},
			expectedErrs: []error{errors.New(`invalid properties.fqdn ""`)},
		},
		"test external only false - invalid fqdn fails": {
			f:            func(oc *api.OpenShiftManagedCluster) { oc.Properties.FQDN = "()" },
			expectedErrs: []error{errors.New(`invalid properties.fqdn "()"`)},
		},
		"provisioning state bad": {
			f:            func(oc *api.OpenShiftManagedCluster) { oc.Properties.ProvisioningState = "bad" },
			expectedErrs: []error{errors.New(`invalid properties.provisioningState "bad"`)},
		},
		"provisioning state Creating": {
			f: func(oc *api.OpenShiftManagedCluster) { oc.Properties.ProvisioningState = "Creating" },
		},
		"provisioning state Failed": {
			f: func(oc *api.OpenShiftManagedCluster) { oc.Properties.ProvisioningState = "Failed" },
		},
		"provisioning state Updating": {
			f: func(oc *api.OpenShiftManagedCluster) { oc.Properties.ProvisioningState = "Updating" },
		},
		"provisioning state Succeeded": {
			f: func(oc *api.OpenShiftManagedCluster) { oc.Properties.ProvisioningState = "Succeeded" },
		},
		"provisioning state Deleting": {
			f: func(oc *api.OpenShiftManagedCluster) { oc.Properties.ProvisioningState = "Deleting" },
		},
		"provisioning state Migrating": {
			f: func(oc *api.OpenShiftManagedCluster) { oc.Properties.ProvisioningState = "Migrating" },
		},
		"provisioning state Upgrading": {
			f: func(oc *api.OpenShiftManagedCluster) { oc.Properties.ProvisioningState = "Upgrading" },
		},
		"provisioning state empty": {
			f: func(oc *api.OpenShiftManagedCluster) { oc.Properties.ProvisioningState = "" },
		},
		"openshift version good": {
			f: func(oc *api.OpenShiftManagedCluster) { oc.Properties.OpenShiftVersion = "v3.11" },
		},
		"openshift version bad": {
			f:            func(oc *api.OpenShiftManagedCluster) { oc.Properties.OpenShiftVersion = "" },
			expectedErrs: []error{errors.New(`invalid properties.openShiftVersion ""`)},
		},
		"openshift config empty public hostname": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.PublicHostname = ""
			},
			expectedErrs: []error{errors.New(`invalid properties.publicHostname ""`)},
		},
		"openshift config invalid public hostname": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.PublicHostname = "bad!"
			},
			expectedErrs: []error{errors.New(`invalid properties.publicHostname "bad!"`)},
		},
		"openshift config matching public hostname": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.PublicHostname = oc.Properties.FQDN
			},
			expectedErrs: []error{errors.New(`invalid properties.fqdn "example.eastus.cloudapp.azure.com": must differ from properties.publicHostname`)},
		},
		"network profile valid VnetId": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.NetworkProfile.VnetID = "/subscriptions/b07e8fae-2f3f-4769-8fa8-8570b426ba13/resourceGroups/test/providers/Microsoft.Network/virtualNetworks/vnet"
			},
		},
		"network profile invalid VnetId": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.NetworkProfile.VnetID = "foo"
			},
			expectedErrs: []error{errors.New(`invalid properties.networkProfile.vnetId "foo"`)},
		},
		"network profile bad vnetCidr": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.NetworkProfile.VnetCIDR = "foo"
			},
			expectedErrs: []error{errors.New(`invalid properties.networkProfile.vnetCidr "foo"`)},
		},
		"network profile invalid vnetCidr": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.NetworkProfile.VnetCIDR = "192.168.0.0/16"
			},
			expectedErrs: []error{
				errors.New(`invalid properties.networkProfile.managementSubnetCIDR "10.0.1.0/24": not contained in properties.networkProfile.vnetCidr "192.168.0.0/16"`),
				errors.New(`invalid properties.agentPoolProfiles["master"].subnetCidr "10.0.0.0/24": not contained in properties.networkProfile.vnetCidr "192.168.0.0/16"`),
				errors.New(`invalid properties.agentPoolProfiles["infra"].subnetCidr "10.0.0.0/24": not contained in properties.networkProfile.vnetCidr "192.168.0.0/16"`),
				errors.New(`invalid properties.agentPoolProfiles["mycompute"].subnetCidr "10.0.0.0/24": not contained in properties.networkProfile.vnetCidr "192.168.0.0/16"`),
			},
		},
		"network profile nil peerVnetId": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.NetworkProfile.PeerVnetID = nil
			},
		},
		"network profile valid peerVnetId": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.NetworkProfile.PeerVnetID = to.StringPtr("/subscriptions/b07e8fae-2f3f-4769-8fa8-8570b426ba13/resourceGroups/test/providers/Microsoft.Network/virtualNetworks/vnet")
			},
		},
		"network profile invalid peerVnetId": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.NetworkProfile.PeerVnetID = to.StringPtr("foo")
			},
			expectedErrs: []error{errors.New(`invalid properties.networkProfile.peerVnetId "foo"`)},
		},
		"network profile invalid managementSubnetCIDR": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.NetworkProfile.ManagementSubnetCIDR = to.StringPtr("foo")
			},
			expectedErrs: []error{errors.New(`invalid properties.networkProfile.managementSubnetCIDR "foo"`)},
		},
		"network profile managementSubnetCIDR - nil - not allowed for private cluster": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.NetworkProfile.ManagementSubnetCIDR = nil
				oc.Properties.PrivateAPIServer = true
			},
			expectedErrs: []error{errors.New(`properties.networkProfile.managementSubnetCIDR cannot be nil with privateAPIServer enabled`)},
		},
		"network profile managementSubnetCIDR - nil - allowed": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.NetworkProfile.ManagementSubnetCIDR = nil
			},
			expectedErrs: nil,
		},
		"router profile duplicate names": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.RouterProfiles =
					append(oc.Properties.RouterProfiles,
						oc.Properties.RouterProfiles[0])
			},
			expectedErrs: []error{errors.New(`duplicate properties.routerProfiles["default"]`)},
		},
		"test external only false - router profile invalid name": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.RouterProfiles[0].Name = "foo"
			},
			// two errors expected here because we require the default profile
			expectedErrs: []error{errors.New(`invalid properties.routerProfiles["foo"]`),
				errors.New(`invalid properties.routerProfiles["default"]`)},
		},
		"test external only true - router profile invalid name": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.RouterProfiles[0].Name = "foo"
			},
			externalOnly: true,
			expectedErrs: []error{errors.New(`invalid properties.routerProfiles["foo"]`)},
		},
		"router profile empty name": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.RouterProfiles[0].Name = ""
			},
			// same as above with 2 errors but additional validate on the individual profile yeilds a third
			// this is not very user friendly but testing as is for now
			// TODO fix
			expectedErrs: []error{errors.New(`invalid properties.routerProfiles[""]`),
				errors.New(`invalid properties.routerProfiles[""]`),
				errors.New(`invalid properties.routerProfiles["default"]`)},
		},
		"router empty public subdomain": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.RouterProfiles[0].PublicSubdomain = ""
			},
			expectedErrs: []error{errors.New(`invalid properties.routerProfiles["default"].publicSubdomain ""`)},
		},
		"router invalid public subdomain": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.RouterProfiles[0].PublicSubdomain = "()"
			},
			expectedErrs: []error{errors.New(`invalid properties.routerProfiles["default"].publicSubdomain "()"`)},
		},
		"router matching public subdomain": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.RouterProfiles[0].PublicSubdomain = oc.Properties.RouterProfiles[0].FQDN
			},
			expectedErrs: []error{errors.New(`invalid properties.routerProfiles["default"].fqdn "router-fqdn.eastus.cloudapp.azure.com": must differ from properties.routerProfiles["default"].publicSubdomain`)},
		},
		"test external only true - unset router profile does not fail": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.RouterProfiles = nil
			},
			externalOnly: true,
		},
		"test external only false - unset router profile does fail": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.RouterProfiles = nil
			},
			expectedErrs: []error{errors.New(`invalid properties.routerProfiles["default"]`)},
		},
		"test external only false - invalid router profile does fail": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.RouterProfiles[0].FQDN = "()"
			},
			expectedErrs: []error{errors.New(`invalid properties.routerProfiles["default"].fqdn "()"`)},
		},
		"agent pool profile duplicate name": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.AgentPoolProfiles = append(
					oc.Properties.AgentPoolProfiles,
					oc.Properties.AgentPoolProfiles[1])
			},
			expectedErrs: []error{errors.New(`duplicate role "infra" in properties.agentPoolProfiles["infra"]`)},
		},
		"agent pool profile invalid infra name": {
			f: func(oc *api.OpenShiftManagedCluster) {
				for i, app := range oc.Properties.AgentPoolProfiles {
					if app.Role == api.AgentPoolProfileRoleInfra {
						oc.Properties.AgentPoolProfiles[i].Name = "foo"
					}
				}
			},
			expectedErrs: []error{
				errors.New(`invalid properties.agentPoolProfiles["foo"]`),
			},
		},
		"agent pool profile invalid compute name": {
			f: func(oc *api.OpenShiftManagedCluster) {
				for i, app := range oc.Properties.AgentPoolProfiles {
					if app.Role == api.AgentPoolProfileRoleCompute {
						oc.Properties.AgentPoolProfiles[i].Name = "$"
					}
				}
			},
			expectedErrs: []error{
				errors.New(`invalid properties.agentPoolProfiles["$"].name "$"`),
			},
		},
		"agent pool profile invalid compute name case": {
			f: func(oc *api.OpenShiftManagedCluster) {
				for i, app := range oc.Properties.AgentPoolProfiles {
					if app.Role == api.AgentPoolProfileRoleCompute {
						oc.Properties.AgentPoolProfiles[i].Name = "UPPERCASE"
					}
				}
			},
			expectedErrs: []error{
				errors.New(`invalid properties.agentPoolProfiles["UPPERCASE"].name "UPPERCASE"`),
			},
		},
		"agent pool profile invalid vm size": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.AgentPoolProfiles[1].VMSize = api.VMSize("SuperBigVM")
			},
			expectedErrs: []error{
				errors.New(`invalid properties.agentPoolProfiles.vmSize "SuperBigVM": master and infra vmSizes must match`),
				errors.New(`invalid properties.agentPoolProfiles["infra"].vmSize "SuperBigVM"`),
			},
		},
		"agent pool unmatched subnet cidr": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.AgentPoolProfiles[2].SubnetCIDR = "10.0.2.0/24"
			},
			expectedErrs: []error{errors.New(`invalid properties.agentPoolProfiles.subnetCidr "10.0.2.0/24": all subnetCidrs must match`)},
		},
		"agent pool bad subnet cidr": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.AgentPoolProfiles[2].SubnetCIDR = "foo"
			},
			expectedErrs: []error{
				errors.New(`invalid properties.agentPoolProfiles.subnetCidr "foo": all subnetCidrs must match`),
				errors.New(`invalid properties.agentPoolProfiles["mycompute"].subnetCidr "foo"`),
			},
		},
		"agent pool subnet cidr clash cluster": {
			f: func(oc *api.OpenShiftManagedCluster) {
				for i := range oc.Properties.AgentPoolProfiles {
					oc.Properties.AgentPoolProfiles[i].SubnetCIDR = "10.128.0.0/24"
				}
			},
			expectedErrs: []error{
				errors.New(`invalid properties.agentPoolProfiles["master"].subnetCidr "10.128.0.0/24": overlaps with cluster network "10.128.0.0/14"`),
				errors.New(`invalid properties.agentPoolProfiles["infra"].subnetCidr "10.128.0.0/24": overlaps with cluster network "10.128.0.0/14"`),
				errors.New(`invalid properties.agentPoolProfiles["mycompute"].subnetCidr "10.128.0.0/24": overlaps with cluster network "10.128.0.0/14"`),
			},
		},
		"agent pool subnet cidr clash service": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.NetworkProfile.VnetCIDR = "172.0.0.0/8"
				for i := range oc.Properties.AgentPoolProfiles {
					oc.Properties.AgentPoolProfiles[i].SubnetCIDR = "172.30.0.0/16"
				}
			},
			expectedErrs: []error{
				errors.New(`invalid properties.networkProfile.managementSubnetCIDR "10.0.1.0/24": not contained in properties.networkProfile.vnetCidr "172.0.0.0/8"`),
				errors.New(`invalid properties.agentPoolProfiles["master"].subnetCidr "172.30.0.0/16": overlaps with service network "172.30.0.0/16"`),
				errors.New(`invalid properties.agentPoolProfiles["infra"].subnetCidr "172.30.0.0/16": overlaps with service network "172.30.0.0/16"`),
				errors.New(`invalid properties.agentPoolProfiles["mycompute"].subnetCidr "172.30.0.0/16": overlaps with service network "172.30.0.0/16"`),
			},
		},
		"agent pool bad master count": {
			f: func(oc *api.OpenShiftManagedCluster) {
				for i, app := range oc.Properties.AgentPoolProfiles {
					if app.Role == api.AgentPoolProfileRoleMaster {
						oc.Properties.AgentPoolProfiles[i].Count = 1
					}
				}
			},
			expectedErrs: []error{errors.New(`invalid properties.masterPoolProfile.count 1`)},
		},
		//we dont check authProfile because it is non pointer struct. Which is all zero values.
		"authProfile.identityProviders empty": {
			f:            func(oc *api.OpenShiftManagedCluster) { oc.Properties.AuthProfile = api.AuthProfile{} },
			expectedErrs: []error{errors.New(`invalid properties.authProfile.identityProviders length`)},
		},
		"AADIdentityProvider secret empty": {
			f: func(oc *api.OpenShiftManagedCluster) {
				aadIdentityProvider := &api.AADIdentityProvider{
					Kind:     "AADIdentityProvider",
					ClientID: "00000000-0000-0000-0000-000000000000",
					Secret:   "",
					TenantID: "00000000-0000-0000-0000-000000000000",
				}
				oc.Properties.AuthProfile.IdentityProviders[0].Provider = aadIdentityProvider
				oc.Properties.AuthProfile.IdentityProviders[0].Name = "Azure AD"
			},
			expectedErrs: []error{errors.New(`invalid properties.authProfile.identityProviders["Azure AD"].secret "<hidden>"`)},
		},
		"AADIdentityProvider clientId empty": {
			f: func(oc *api.OpenShiftManagedCluster) {
				aadIdentityProvider := &api.AADIdentityProvider{
					Kind:     "AADIdentityProvider",
					ClientID: "",
					Secret:   "aadClientSecret",
					TenantID: "00000000-0000-0000-0000-000000000000",
				}
				oc.Properties.AuthProfile.IdentityProviders[0].Provider = aadIdentityProvider
				oc.Properties.AuthProfile.IdentityProviders[0].Name = "Azure AD"
			},
			expectedErrs: []error{errors.New(`invalid properties.authProfile.identityProviders["Azure AD"].clientId ""`)},
		},
		"AADIdentityProvider tenantId empty": {
			f: func(oc *api.OpenShiftManagedCluster) {
				aadIdentityProvider := &api.AADIdentityProvider{
					Kind:     "AADIdentityProvider",
					ClientID: "00000000-0000-0000-0000-000000000000",
					Secret:   "aadClientSecret",
					TenantID: "",
				}
				oc.Properties.AuthProfile.IdentityProviders[0].Provider = aadIdentityProvider
				oc.Properties.AuthProfile.IdentityProviders[0].Name = "Azure AD"
			},
			expectedErrs: []error{errors.New(`invalid properties.authProfile.identityProviders["Azure AD"].tenantId ""`)},
		},
		"AADIdentityProvider kind empty": {
			f: func(oc *api.OpenShiftManagedCluster) {
				aadIdentityProvider := &api.AADIdentityProvider{
					Kind:     "",
					ClientID: "00000000-0000-0000-0000-000000000000",
					Secret:   "aadClientSecret",
					TenantID: "00000000-0000-0000-0000-000000000000",
				}
				oc.Properties.AuthProfile.IdentityProviders[0].Provider = aadIdentityProvider
				oc.Properties.AuthProfile.IdentityProviders[0].Name = "Azure AD"
			},
			expectedErrs: []error{errors.New(`invalid properties.authProfile.identityProviders["Azure AD"].kind ""`)},
		},
		"master service principal bad client": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.MasterServicePrincipalProfile.ClientID = "bad"
			},
			expectedErrs: []error{errors.New(`invalid properties.masterServicePrincipalProfile.clientID "bad"`)},
		},
		"master service principal empty secret": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.MasterServicePrincipalProfile.Secret = ""
			},
			expectedErrs: []error{errors.New(`invalid properties.masterServicePrincipalProfile.secret ""`)},
		},
		"worker service principal bad client": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.WorkerServicePrincipalProfile.ClientID = "bad"
			},
			expectedErrs: []error{errors.New(`invalid properties.workerServicePrincipalProfile.clientID "bad"`)},
		},
		"worker service principal empty secret": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.WorkerServicePrincipalProfile.Secret = ""
			},
			expectedErrs: []error{errors.New(`invalid properties.workerServicePrincipalProfile.secret ""`)},
		},
		"azprofile empty resource group": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.AzProfile.ResourceGroup = ""
			},
			expectedErrs: []error{errors.New(`invalid properties.azProfile.resourceGroup ""`)},
		},
		"azprofile bad resource group": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.AzProfile.ResourceGroup = "bad!"
			},
			expectedErrs: []error{errors.New(`invalid properties.azProfile.resourceGroup "bad!"`)},
		},
		"azprofile bad tenantid": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.AzProfile.TenantID = "bad"
			},
			expectedErrs: []error{errors.New(`invalid properties.azProfile.tenantId "bad"`)},
		},
		"azprofile bad subscriptionid": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.AzProfile.SubscriptionID = ""
			},
			expectedErrs: []error{errors.New(`invalid properties.azProfile.subscriptionId ""`)},
		},
		"routercertprofile bad secreturl": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.RouterProfiles[0].RouterCertProfile.KeyVaultSecretURL = "bad"
			},
			expectedErrs: []error{errors.New(`invalid properties.routerProfiles["default"].routerCertProfile.keyVaultSecretURL "bad"`)},
		},
		"apicertprofile bad secreturl": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.APICertProfile.KeyVaultSecretURL = "bad"
			},
			expectedErrs: []error{errors.New(`invalid properties.apiCertProfile.keyVaultSecretURL "bad"`)},
		},
		"monitorProfile missing configs when enabled": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.MonitorProfile.WorkspaceID = ""
				oc.Properties.MonitorProfile.WorkspaceKey = ""
				oc.Properties.MonitorProfile.WorkspaceResourceID = ""
			},
			expectedErrs: []error{
				errors.New(`properties.monitorProfile.workspaceResourceID cannot be empty if properties.monitorProfile.enabled = true`),
				errors.New(`properties.monitorProfile.workspaceID cannot be empty if properties.monitorProfile.enabled = true`),
				errors.New(`properties.monitorProfile.workspaceKey cannot be empty if properties.monitorProfile.enabled = true`),
			},
		},
		"monitorProfile bad resource id": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.MonitorProfile.WorkspaceResourceID = "bad!"
			},
			expectedErrs: []error{
				errors.New(`invalid properties.monitorProfile.workspaceResourceId "bad!"`),
			},
		},
		"monitorProfile bad workspace id": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.MonitorProfile.WorkspaceID = "bad!"
			},
			expectedErrs: []error{
				errors.New(`invalid properties.monitorProfile.workspaceId "bad!"`),
			},
		},
		"monitorProfile bad workspace key": {
			f: func(oc *api.OpenShiftManagedCluster) {
				oc.Properties.MonitorProfile.WorkspaceKey = "bad!"
			},
			expectedErrs: []error{
				errors.New(`invalid properties.monitorProfile.workspaceKey`),
			},
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

		{
			v := APIValidator{runningUnderTest: !test.simulateProd}
			errs := v.Validate(cs, cs, test.externalOnly)
			if !reflect.DeepEqual(errs, test.expectedErrs) {
				t.Logf("api test case %q", name)
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

		if !test.externalOnly {
			v := AdminAPIValidator{runningUnderTest: !test.simulateProd}
			errs := v.Validate(cs, cs)
			if !reflect.DeepEqual(errs, test.expectedErrs) {
				t.Logf("admin test case %q", name)
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
}
