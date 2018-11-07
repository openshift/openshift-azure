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
			t.Fatal(err)
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
	cs := api.OpenShiftManagedCluster{
		Properties: &api.Properties{
			RouterProfiles: []api.RouterProfile{
				{
					FQDN: "one.two.three",
				},
			},
		},
	}
	if got := Derived.RouterLBCNamePrefix(&cs); got != "one" {
		t.Errorf("derived.RouterLBCNamePrefix() = %v, want %v", got, "one")
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
		cs        api.OpenShiftManagedCluster
		role      api.AgentPoolProfileRole
		wantKR    string
		wantKRErr string
		wantSR    string
		wantSRErr string
	}{
		{
			cs: api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Role:   api.AgentPoolProfileRoleCompute,
							VMSize: api.StandardD4sV3,
						},
					},
				},
			},
			role:   api.AgentPoolProfileRoleCompute,
			wantKR: "cpu=500m,memory=512Mi",
			wantSR: "cpu=500m,memory=512Mi",
		},
		{
			cs: api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Role:   api.AgentPoolProfileRoleInfra,
							VMSize: api.StandardD2sV3,
						},
					},
				}},
			role:   api.AgentPoolProfileRoleInfra,
			wantKR: "cpu=200m,memory=512Mi",
			wantSR: "cpu=200m,memory=512Mi",
		},
		{
			cs: api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Role:   api.AgentPoolProfileRoleMaster,
							VMSize: api.StandardD2sV3,
						},
					},
				},
			},
			role:      api.AgentPoolProfileRoleMaster,
			wantKRErr: "kubereserved not defined for role master",
			wantSR:    "cpu=500m,memory=1Gi",
		},
		{
			cs: api.OpenShiftManagedCluster{
				Properties: &api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Role:   api.AgentPoolProfileRoleMaster,
							VMSize: api.StandardD4sV3,
						},
					},
				},
			},
			role:      api.AgentPoolProfileRoleMaster,
			wantKRErr: "kubereserved not defined for role master",
			wantSR:    "cpu=1000m,memory=1Gi",
		},
		{
			cs: api.OpenShiftManagedCluster{
				Properties: &api.Properties{},
			},
			role:      "anewrole",
			wantKRErr: "role anewrole not found",
			wantSRErr: "role anewrole not found",
		},
	}
	for _, tt := range tests {
		got, err := Derived.KubeReserved(&tt.cs, tt.role)
		if got != tt.wantKR || (err == nil && tt.wantKRErr != "") || (err != nil && err.Error() != tt.wantKRErr) {
			t.Errorf("derived.KubeReserved(%s) = %v, %v: wanted %v, %v", tt.role, got, err, tt.wantKR, tt.wantKRErr)
		}

		got, err = Derived.SystemReserved(&tt.cs, tt.role)
		if got != tt.wantSR || (err == nil && tt.wantSRErr != "") || (err != nil && err.Error() != tt.wantSRErr) {
			t.Errorf("derived.SystemReserved(%s) = %v, %v: wanted %v, %v", tt.role, got, err, tt.wantSR, tt.wantSRErr)
		}
	}
}
