package api

import (
	"reflect"
	"testing"

	"github.com/openshift/openshift-azure/pkg/api/v1"
)

var testOpenShiftCluster = &v1.OpenShiftCluster{
	ID:       "id",
	Location: "location",
	Name:     "name",
	Plan: &v1.ResourcePurchasePlan{
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
	Properties: &v1.Properties{
		ProvisioningState: "properties.provisioningState",
		OpenShiftVersion:  "properties.openShiftVersion",
		PublicHostname:    "properties.publicHostname",
		FQDN:              "properties.fqdn",
		RouterProfiles: []v1.RouterProfile{
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
		MasterPoolProfile: v1.MasterPoolProfile{
			ProfileSpec: v1.ProfileSpec{
				Name:         "properties.agentPoolProfiles.0.name",
				Count:        1,
				VMSize:       "properties.agentPoolProfiles.0.vmSize",
				VnetSubnetID: "properties.agentPoolProfiles.0.vnetSubnetID",
				OSType:       "properties.agentPoolProfiles.0.osType",
			},
		},
		AgentPoolProfiles: []v1.AgentPoolProfile{
			{
				Role: "properties.agentPoolProfiles.0.role",
				ProfileSpec: v1.ProfileSpec{
					Name:         "properties.agentPoolProfiles.0.name",
					Count:        1,
					VMSize:       "properties.agentPoolProfiles.0.vmSize",
					VnetSubnetID: "properties.agentPoolProfiles.0.vnetSubnetID",
					OSType:       "properties.agentPoolProfiles.0.osType",
				},
			},
			{
				Role: "properties.agentPoolProfiles.0.role",
				ProfileSpec: v1.ProfileSpec{
					Name:         "properties.agentPoolProfiles.0.name",
					Count:        2,
					VMSize:       "properties.agentPoolProfiles.0.vmSize",
					VnetSubnetID: "properties.agentPoolProfiles.0.vnetSubnetID",
					OSType:       "properties.agentPoolProfiles.0.osType",
				},
			},
		},
		ServicePrincipalProfile: v1.ServicePrincipalProfile{
			ClientID: "properties.servicePrincipalProfile.clientID",
			Secret:   "properties.servicePrincipalProfile.secret",
		},
	},
}

var testContainerService = &ContainerService{
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
		OrchestratorProfile: &OrchestratorProfile{
			OrchestratorVersion: "properties.openShiftVersion",
			OpenShiftConfig: &OpenShiftConfig{
				PublicHostname: "properties.publicHostname",
				RouterProfiles: []OpenShiftRouterProfile{
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
			},
		},
		FQDN: "properties.fqdn",
		AgentPoolProfiles: []*AgentPoolProfile{
			{
				Name:         "properties.agentPoolProfiles.0.name",
				Count:        1,
				VMSize:       "properties.agentPoolProfiles.0.vmSize",
				OSType:       "properties.agentPoolProfiles.0.osType",
				VnetSubnetID: "properties.agentPoolProfiles.0.vnetSubnetID",
				Role:         "properties.agentPoolProfiles.0.role",
			},
			{
				Name:         "properties.agentPoolProfiles.0.name",
				Count:        2,
				VMSize:       "properties.agentPoolProfiles.0.vmSize",
				OSType:       "properties.agentPoolProfiles.0.osType",
				VnetSubnetID: "properties.agentPoolProfiles.0.vnetSubnetID",
				Role:         "properties.agentPoolProfiles.0.role",
			},
			{
				Name:         "properties.agentPoolProfiles.0.name",
				Count:        1,
				VMSize:       "properties.agentPoolProfiles.0.vmSize",
				OSType:       "properties.agentPoolProfiles.0.osType",
				VnetSubnetID: "properties.agentPoolProfiles.0.vnetSubnetID",
				Role:         "master",
			},
		},
		ServicePrincipalProfile: &ServicePrincipalProfile{
			ClientID: "properties.servicePrincipalProfile.clientID",
			Secret:   "properties.servicePrincipalProfile.secret",
		},
	},
}

func TestConvertVLabsOpenShiftClusterToContainerService(t *testing.T) {
	cs := ConvertVLabsOpenShiftClusterToContainerService(testOpenShiftCluster)
	if !reflect.DeepEqual(cs, testContainerService) {
		t.Errorf("ConvertVLabsOpenShiftClusterToContainerService returned unexpected result\n%#v\n", cs)
	}
}
