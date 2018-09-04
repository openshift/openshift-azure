package api

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
)

var testOpenShiftCluster = &v20180930preview.OpenShiftManagedCluster{
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
					},
				},
			},
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
			Name:         "properties.agentPoolProfiles.master.name",
			Count:        1,
			VMSize:       "properties.agentPoolProfiles.master.vmSize",
			VnetSubnetID: "properties.agentPoolProfiles.master.vnetSubnetID",
			OSType:       "properties.agentPoolProfiles.master.osType",
		},
		AgentPoolProfiles: []v20180930preview.AgentPoolProfile{
			{
				Role:         "infra",
				Name:         "properties.agentPoolProfiles.infra.name",
				Count:        2,
				VMSize:       "properties.agentPoolProfiles.infra.vmSize",
				VnetSubnetID: "properties.agentPoolProfiles.infra.vnetSubnetID",
				OSType:       "properties.agentPoolProfiles.infra.osType",
			},
			{
				Role:         "compute",
				Name:         "properties.agentPoolProfiles.compute.name",
				Count:        3,
				VMSize:       "properties.agentPoolProfiles.compute.vmSize",
				VnetSubnetID: "properties.agentPoolProfiles.compute.vnetSubnetID",
				OSType:       "properties.agentPoolProfiles.compute.osType",
			},
		},
		ServicePrincipalProfile: &v20180930preview.ServicePrincipalProfile{
			ClientID: "properties.servicePrincipalProfile.clientId",
			Secret:   "properties.servicePrincipalProfile.secret",
		},
	},
}

var testContainerService = &OpenShiftManagedCluster{
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
					},
				},
			},
		},
		AgentPoolProfiles: map[AgentPoolProfileRole]AgentPoolProfile{
			AgentPoolProfileRoleMaster: {
				Name:         "properties.agentPoolProfiles.master.name",
				Count:        1,
				VMSize:       "properties.agentPoolProfiles.master.vmSize",
				OSType:       "properties.agentPoolProfiles.master.osType",
				VnetSubnetID: "properties.agentPoolProfiles.master.vnetSubnetID",
			},
			AgentPoolProfileRoleInfra: {
				Name:         "properties.agentPoolProfiles.infra.name",
				Count:        2,
				VMSize:       "properties.agentPoolProfiles.infra.vmSize",
				OSType:       "properties.agentPoolProfiles.infra.osType",
				VnetSubnetID: "properties.agentPoolProfiles.infra.vnetSubnetID",
			},
			AgentPoolProfileRoleCompute: {
				Name:         "properties.agentPoolProfiles.compute.name",
				Count:        3,
				VMSize:       "properties.agentPoolProfiles.compute.vmSize",
				OSType:       "properties.agentPoolProfiles.compute.osType",
				VnetSubnetID: "properties.agentPoolProfiles.compute.vnetSubnetID",
			},
		},
		ServicePrincipalProfile: &ServicePrincipalProfile{
			ClientID: "properties.servicePrincipalProfile.clientId",
			Secret:   "properties.servicePrincipalProfile.secret",
		},
	},
}

func TestConvertFromV20180930preview(t *testing.T) {
	cs := ConvertFromV20180930preview(testOpenShiftCluster)
	if !reflect.DeepEqual(cs, testContainerService) {
		t.Errorf("ConvertFromV20180930preview returned unexpected result\n%s\n", spew.Sdump(cs))
	}
}
