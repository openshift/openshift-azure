package v1

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

var testOpenShiftCluster = &OpenShiftManagedCluster{
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
		FQDN:              "properties.fqdn",
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
		MasterPoolProfile: MasterPoolProfile{
			ProfileSpec: ProfileSpec{
				Name:         "properties.agentPoolProfiles.0.name",
				Count:        1,
				VMSize:       "properties.agentPoolProfiles.0.vmSize",
				VnetSubnetID: "properties.agentPoolProfiles.0.vnetSubnetID",
				OSType:       "properties.agentPoolProfiles.0.osType",
			},
		},
		AgentPoolProfiles: []AgentPoolProfile{
			{
				ProfileSpec: ProfileSpec{
					Name:         "properties.agentPoolProfiles.0.name",
					Count:        1,
					VMSize:       "properties.agentPoolProfiles.0.vmSize",
					VnetSubnetID: "properties.agentPoolProfiles.0.vnetSubnetID",
					OSType:       "properties.agentPoolProfiles.0.osType",
				},
				Role: "properties.agentPoolProfiles.0.role",
			},
			{
				ProfileSpec: ProfileSpec{
					Name:         "properties.agentPoolProfiles.0.name",
					Count:        2,
					VMSize:       "properties.agentPoolProfiles.0.vmSize",
					VnetSubnetID: "properties.agentPoolProfiles.0.vnetSubnetID",
					OSType:       "properties.agentPoolProfiles.0.osType",
				},
				Role: "properties.agentPoolProfiles.0.role",
			},
		},
		ServicePrincipalProfile: ServicePrincipalProfile{
			ClientID: "properties.servicePrincipalProfile.clientID",
			Secret:   "properties.servicePrincipalProfile.secret",
		},
	},
}

var testOpenShiftClusterJSON = []byte(`{
	"id": "id",
	"location": "location",
	"name": "name",
	"plan": {
		"name": "plan.name",
		"product": "plan.product",
		"promotionCode": "plan.promotionCode",
		"publisher": "plan.publisher"
	},
	"tags": {
		"tags.k1": "v1",
		"tags.k2": "v2"
	},
	"type": "type",
	"properties": {
		"provisioningState": "properties.provisioningState",
		"openShiftVersion": "properties.openShiftVersion",
		"publicHostname": "properties.publicHostname",
		"fqdn": "properties.fqdn",
		"routerProfiles": [
			{
				"name": "properties.routerProfiles.0.name",
				"publicSubdomain": "properties.routerProfiles.0.publicSubdomain",
				"fqdn": "properties.routerProfiles.0.fqdn"
			},
			{
				"name": "properties.routerProfiles.1.name",
				"publicSubdomain": "properties.routerProfiles.1.publicSubdomain",
				"fqdn": "properties.routerProfiles.1.fqdn"
			}
		],
		"masterPoolProfile": {
			"name": "properties.agentPoolProfiles.0.name",
			"count": 1,
			"vmSize": "properties.agentPoolProfiles.0.vmSize",
			"vnetSubnetID": "properties.agentPoolProfiles.0.vnetSubnetID",
			"osType": "properties.agentPoolProfiles.0.osType"
		},
		"agentPoolProfiles": [
			{
				"name": "properties.agentPoolProfiles.0.name",
				"count": 1,
				"vmSize": "properties.agentPoolProfiles.0.vmSize",
				"vnetSubnetID": "properties.agentPoolProfiles.0.vnetSubnetID",
				"osType": "properties.agentPoolProfiles.0.osType",
				"role": "properties.agentPoolProfiles.0.role"
			},
			{
				"name": "properties.agentPoolProfiles.0.name",
				"count": 2,
				"vmSize": "properties.agentPoolProfiles.0.vmSize",
				"vnetSubnetID": "properties.agentPoolProfiles.0.vnetSubnetID",
				"osType": "properties.agentPoolProfiles.0.osType",
				"role": "properties.agentPoolProfiles.0.role"
			}
		],
		"servicePrincipalProfile": {
			"clientId": "properties.servicePrincipalProfile.clientID",
			"secret": "properties.servicePrincipalProfile.secret"
		}
	}
}`)

func TestMarshal(t *testing.T) {
	b, err := json.MarshalIndent(testOpenShiftCluster, "", "\t")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, testOpenShiftClusterJSON) {
		t.Errorf("json.MarshalIndent returned unexpected result\n%s\n", string(b))
	}
}

func TestUnmarshal(t *testing.T) {
	var oc *OpenShiftManagedCluster
	err := json.Unmarshal(testOpenShiftClusterJSON, &oc)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(oc, testOpenShiftCluster) {
		t.Errorf("json.Unmarshal returned unexpected result\n%#v\n", oc)
	}
}
