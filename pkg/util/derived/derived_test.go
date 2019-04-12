package derived

import (
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestDerivedCloudProviderConf(t *testing.T) {
	tests := []struct {
		cs         api.OpenShiftManagedCluster
		name       string
		wantMaster []byte
		wantWorker []byte
	}{
		{
			name: "one",
			cs: api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AzProfile: api.AzProfile{
						TenantID:       "tenant",
						SubscriptionID: "sub",
						ResourceGroup:  "rg",
					},
					MasterServicePrincipalProfile: api.ServicePrincipalProfile{
						ClientID: "master_client_id",
						Secret:   "master_client_secrett",
					},
					WorkerServicePrincipalProfile: api.ServicePrincipalProfile{
						ClientID: "worker_client_id",
						Secret:   "worker_client_secrett",
					},
					AgentPoolProfiles: []api.AgentPoolProfile{
						{Role: api.AgentPoolProfileRoleMaster, Name: "master"},
						{Role: api.AgentPoolProfileRoleInfra, Name: "infra"},
						{Role: api.AgentPoolProfileRoleCompute, Name: "computetest"},
					},
				},
				Location: "eastus",
			},
			wantMaster: []byte(`aadClientId: master_client_id
aadClientSecret: master_client_secrett
cloudProviderBackoff: true
cloudProviderBackoffDuration: 5
cloudProviderBackoffExponent: 1.5
cloudProviderBackoffJitter: 1
cloudProviderBackoffRetries: 6
cloudProviderRateLimit: true
cloudProviderRateLimitBucket: 10
cloudProviderRateLimitQPS: 3
loadBalancerSku: standard
location: eastus
resourceGroup: rg
securityGroupName: nsg-worker
subnetName: default
subscriptionId: sub
tenantId: tenant
useInstanceMetadata: true
vmType: vmss
vnetName: vnet
`),
			wantWorker: []byte(`aadClientId: worker_client_id
aadClientSecret: worker_client_secrett
cloudProviderBackoff: true
cloudProviderBackoffDuration: 5
cloudProviderBackoffExponent: 1.5
cloudProviderBackoffJitter: 1
cloudProviderBackoffRetries: 6
cloudProviderRateLimit: true
cloudProviderRateLimitBucket: 10
cloudProviderRateLimitQPS: 3
loadBalancerSku: standard
location: eastus
resourceGroup: rg
securityGroupName: nsg-worker
subnetName: default
subscriptionId: sub
tenantId: tenant
useInstanceMetadata: true
vmType: vmss
vnetName: vnet
`),
		},
	}

	for _, tt := range tests {
		got, err := MasterCloudProviderConf(&tt.cs, true, true)
		if err != nil {
			t.Fatal(err)
			return
		}
		if !reflect.DeepEqual(got, tt.wantMaster) {
			t.Errorf("derived.MasterCloudProviderConf() = \"%v\", want \"%v\"", string(got), string(tt.wantMaster))
		}
		got, err = WorkerCloudProviderConf(&tt.cs, true, true)
		if err != nil {
			t.Fatal(err)
			return
		}
		if !reflect.DeepEqual(got, tt.wantWorker) {
			t.Errorf("derived.WorkerCloudProviderConf() = \"%v\", want \"%v\"", string(got), string(tt.wantWorker))
		}
	}
}

func TestDerivedAADGroupSyncConf(t *testing.T) {
	provider := api.AADIdentityProvider{
		ClientID:             "client_id",
		Secret:               "hush",
		TenantID:             "tenant-id",
		CustomerAdminGroupID: to.StringPtr("customerAdminGroupId"),
	}

	cs := api.OpenShiftManagedCluster{
		Properties: api.Properties{
			AuthProfile: api.AuthProfile{
				IdentityProviders: []api.IdentityProvider{
					{
						Provider: &provider,
					},
				},
			},
		},
	}
	want := []byte(`clientId: client_id
customerAdminGroupId: customerAdminGroupId
secret: hush
tenantId: tenant-id
`)

	got, err := AadGroupSyncConf(&cs)
	if err != nil {
		t.Fatal(err)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("derived.AadGroupSyncConf() = \"%v\", want \"%v\"", string(got), string(want))
	}
}
