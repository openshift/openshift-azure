package config

import (
	"reflect"
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestDerivedCloudProviderConf(t *testing.T) {
	tests := []struct {
		cs   api.OpenShiftManagedCluster
		name string
		want []byte
	}{
		{
			name: "one",
			cs: api.OpenShiftManagedCluster{
				Properties: &api.Properties{AzProfile: &api.AzProfile{
					TenantID:       "tenant",
					SubscriptionID: "sub",
					ResourceGroup:  "rg",
				}, ServicePrincipalProfile: &api.ServicePrincipalProfile{
					ClientID: "client_id",
					Secret:   "client_secrett",
				}},
				Location: "eastus",
			},
			want: []byte(`aadClientId: client_id
aadClientSecret: client_secrett
location: eastus
primaryScaleSetName: ss-compute
resourceGroup: rg
securityGroupName: nsg-compute
subscriptionId: sub
tenantId: tenant
vmType: vmss
`),
		},
	}

	for _, tt := range tests {
		got, err := Derived.CloudProviderConf(&tt.cs)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("derived.CloudProviderConf() = \"%v\", want \"%v\"", string(got), string(tt.want))
		}
	}
}

func TestDerivedMasterLBCNamePrefix(t *testing.T) {
	cs := api.OpenShiftManagedCluster{
		Properties: &api.Properties{FQDN: "bar.baz"},
	}
	if got := Derived.MasterLBCNamePrefix(&cs); got != "bar" {
		t.Errorf("derived.MasterLBCNamePrefix() = %v, want %v", got, "bar")
	}
}

func TestDerivedRouterLBCNamePrefix(t *testing.T) {
	tests := []struct {
		cs   api.OpenShiftManagedCluster
		want string
	}{
		{
			cs: api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					RouterProfiles: []api.RouterProfile{
						{
							FQDN: "one.two.three",
						},
						{
							FQDN: "not.this",
						},
					},
				},
			},
			want: "one",
		},
	}
	for _, tt := range tests {
		if got := Derived.RouterLBCNamePrefix(&tt.cs); got != tt.want {
			t.Errorf("derived.RouterLBCNamePrefix() = %v, want %v", got, tt.want)
		}
	}
}

func TestDerivedPublicHostname(t *testing.T) {
	tests := []struct {
		cs   api.OpenShiftManagedCluster
		want string
	}{
		{
			cs: api.OpenShiftManagedCluster{
				Properties: &api.Properties{FQDN: "bar", PublicHostname: "baar"},
			},
			want: "baar",
		},
		{
			cs: api.OpenShiftManagedCluster{
				Properties: &api.Properties{FQDN: "bar", PublicHostname: ""},
			},
			want: "bar",
		},
	}
	for _, tt := range tests {
		if got := Derived.PublicHostname(&tt.cs); got != tt.want {
			t.Errorf("derived.PublicHostname() = %v, want %v", got, tt.want)
		}
	}
}

func TestDerivedKubeAndSystemReserved(t *testing.T) {
	tests := []struct {
		cs   api.OpenShiftManagedCluster
		role api.AgentPoolProfileRole
		want string
	}{
		{
			cs: api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Name:   "1",
							Role:   api.AgentPoolProfileRoleCompute,
							VMSize: "Standard_D2s_v3",
						},
					},
				},
			},
			want: "cpu=200m,memory=512Mi",
			role: api.AgentPoolProfileRoleCompute,
		},
		{
			cs: api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Name:   "1",
							Role:   api.AgentPoolProfileRoleCompute,
							VMSize: "Standard_D2s_v3",
						},
						{
							Name:   "2",
							Role:   api.AgentPoolProfileRoleInfra,
							VMSize: "Standard_D2s_v3",
						},
					},
				}},
			want: "cpu=200m,memory=512Mi",
			role: api.AgentPoolProfileRoleInfra,
		},
		{
			cs: api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Name:   "infra",
							Role:   api.AgentPoolProfileRoleMaster,
							VMSize: "unknown",
						},
					},
				},
			},
			want: "",
			role: api.AgentPoolProfileRoleMaster,
		},
		{
			cs: api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{},
				},
			},
			want: "",
			role: "anewrole",
		},
	}
	for _, tt := range tests {
		if got := Derived.KubeReserved(&tt.cs, tt.role); got != tt.want {
			t.Errorf("derived.KubeReserved(%s) = %v, want %v", tt.role, got, tt.want)
		}

		if got := Derived.SystemReserved(&tt.cs, tt.role); got != tt.want {
			t.Errorf("derived.SystemReserved(%s) = %v, want %v", tt.role, got, tt.want)
		}
	}
}
