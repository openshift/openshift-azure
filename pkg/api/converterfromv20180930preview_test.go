package api

import (
	"reflect"
	"testing"

	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
)

var v20180930previewManagedCluster = &v20180930preview.OpenShiftManagedCluster{
	ID:       "id",
	Location: "location",
	Name:     "name",
	Plan: &v20180930preview.ResourcePurchasePlan{
		Name:          "plan.name",
		Product:       "plan.product",
		PromotionCode: "plan.promotionCode",
		Publisher:     "plan.publisher",
	},
	Tags: map[string]string{
		"tags.k1": "v1",
		"tags.k2": "v2",
	},
	Type: "type",
	Properties: &v20180930preview.Properties{
		ProvisioningState: "properties.provisioningState",
		OpenShiftVersion:  "properties.openShiftVersion",
		PublicHostname:    "properties.publicHostname",
		FQDN:              "properties.fqdn",
		AuthProfile: &v20180930preview.AuthProfile{
			IdentityProviders: []v20180930preview.IdentityProvider{
				{
					Name: "properties.authProfile.identityProviders.0.name",
					Provider: &v20180930preview.AADIdentityProvider{
						Kind:     "AADIdentityProvider",
						ClientID: "properties.authProfile.identityProviders.0.provider.clientId",
						Secret:   "properties.authProfile.identityProviders.0.provider.secret",
						TenantID: "properties.authProfile.identityProviders.0.provider.tenantId",
					},
				},
			},
		},
		NetworkProfile: &v20180930preview.NetworkProfile{
			VnetCIDR:   "properties.networkProfile.vnetCidr",
			PeerVnetID: "properties.networkProfile.peerVnetId",
		},
		RouterProfiles: []v20180930preview.RouterProfile{
			{
				Name:            "properties.routerProfiles.0.name",
				PublicSubdomain: "properties.routerProfiles.0.publicSubdomain",
				FQDN:            "properties.routerProfiles.0.fqdn",
			},
			{
				Name:            "properties.routerProfiles.1.name",
				PublicSubdomain: "properties.routerProfiles.1.publicSubdomain",
				FQDN:            "properties.routerProfiles.1.fqdn",
			},
		},
		MasterPoolProfile: &v20180930preview.MasterPoolProfile{
			Count:      1,
			VMSize:     "properties.agentPoolProfiles.0.vmSize",
			SubnetCIDR: "properties.agentPoolProfiles.0.subnetCidr",
		},
		AgentPoolProfiles: []v20180930preview.AgentPoolProfile{
			{
				Role:       "properties.agentPoolProfiles.0.role",
				Name:       "properties.agentPoolProfiles.0.name",
				Count:      1,
				VMSize:     "properties.agentPoolProfiles.0.vmSize",
				SubnetCIDR: "properties.agentPoolProfiles.0.subnetCidr",
				OSType:     "properties.agentPoolProfiles.0.osType",
			},
			{
				Role:       "properties.agentPoolProfiles.0.role",
				Name:       "properties.agentPoolProfiles.0.name",
				Count:      2,
				VMSize:     "properties.agentPoolProfiles.0.vmSize",
				SubnetCIDR: "properties.agentPoolProfiles.0.subnetCidr",
				OSType:     "properties.agentPoolProfiles.0.osType",
			},
		},
	},
}

var internalManagedCluster = &OpenShiftManagedCluster{
	ID:       "id",
	Location: "location",
	Name:     "name",
	Plan: &ResourcePurchasePlan{
		Name:          "plan.name",
		Product:       "plan.product",
		PromotionCode: "plan.promotionCode",
		Publisher:     "plan.publisher",
	},
	Tags: map[string]string{
		"tags.k1": "v1",
		"tags.k2": "v2",
	},
	Type: "type",
	Properties: &Properties{
		ProvisioningState: "properties.provisioningState",
		OpenShiftVersion:  "properties.openShiftVersion",
		PublicHostname:    "properties.publicHostname",
		RouterProfiles: []RouterProfile{
			{
				Name:            "properties.routerProfiles.0.name",
				PublicSubdomain: "properties.routerProfiles.0.publicSubdomain",
				FQDN:            "properties.routerProfiles.0.fqdn",
			},
			{
				Name:            "properties.routerProfiles.1.name",
				PublicSubdomain: "properties.routerProfiles.1.publicSubdomain",
				FQDN:            "properties.routerProfiles.1.fqdn",
			},
		},
		FQDN: "properties.fqdn",
		AuthProfile: &AuthProfile{
			IdentityProviders: []IdentityProvider{
				{
					Name: "properties.authProfile.identityProviders.0.name",
					Provider: &AADIdentityProvider{
						Kind:     "AADIdentityProvider",
						ClientID: "properties.authProfile.identityProviders.0.provider.clientId",
						Secret:   "properties.authProfile.identityProviders.0.provider.secret",
						TenantID: "properties.authProfile.identityProviders.0.provider.tenantId",
					},
				},
			},
		},
		NetworkProfile: &NetworkProfile{
			VnetCIDR:   "properties.networkProfile.vnetCidr",
			PeerVnetID: "properties.networkProfile.peerVnetId",
		},
		AgentPoolProfiles: []AgentPoolProfile{
			{
				Name:       "properties.agentPoolProfiles.0.name",
				Count:      1,
				VMSize:     "properties.agentPoolProfiles.0.vmSize",
				SubnetCIDR: "properties.agentPoolProfiles.0.subnetCidr",
				OSType:     "properties.agentPoolProfiles.0.osType",
				Role:       "properties.agentPoolProfiles.0.role",
			},
			{
				Name:       "properties.agentPoolProfiles.0.name",
				Count:      2,
				VMSize:     "properties.agentPoolProfiles.0.vmSize",
				SubnetCIDR: "properties.agentPoolProfiles.0.subnetCidr",
				OSType:     "properties.agentPoolProfiles.0.osType",
				Role:       "properties.agentPoolProfiles.0.role",
			},
			{
				Name:       string(AgentPoolProfileRoleMaster),
				Count:      1,
				VMSize:     "properties.agentPoolProfiles.0.vmSize",
				SubnetCIDR: "properties.agentPoolProfiles.0.subnetCidr",
				OSType:     OSTypeLinux,
				Role:       AgentPoolProfileRoleMaster,
			},
		},
	},
}

func TestConvertFromV20180930preview(t *testing.T) {
	cs := ConvertFromV20180930preview(v20180930previewManagedCluster)
	if !reflect.DeepEqual(cs, internalManagedCluster) {
		t.Errorf("ConvertFromV20180930preview returned unexpected result\n%#v\n", cs)
	}
}
