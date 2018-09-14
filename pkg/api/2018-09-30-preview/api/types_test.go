package api

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

var unmarshalled = &OpenShiftManagedCluster{
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
		MasterPoolProfile: &MasterPoolProfile{
			Count:        1,
			VMSize:       "properties.agentPoolProfiles.0.vmSize",
			VnetSubnetID: "properties.agentPoolProfiles.0.vnetSubnetID",
		},
		AgentPoolProfiles: []AgentPoolProfile{
			{
				Name:         "properties.agentPoolProfiles.0.name",
				Count:        1,
				VMSize:       "properties.agentPoolProfiles.0.vmSize",
				VnetSubnetID: "properties.agentPoolProfiles.0.vnetSubnetID",
				OSType:       "properties.agentPoolProfiles.0.osType",
				Role:         "properties.agentPoolProfiles.0.role",
			},
			{
				Name:         "properties.agentPoolProfiles.0.name",
				Count:        2,
				VMSize:       "properties.agentPoolProfiles.0.vmSize",
				VnetSubnetID: "properties.agentPoolProfiles.0.vnetSubnetID",
				OSType:       "properties.agentPoolProfiles.0.osType",
				Role:         "properties.agentPoolProfiles.0.role",
			},
		},
	},
}

var marshalled = []byte(`{
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
			"count": 1,
			"vmSize": "properties.agentPoolProfiles.0.vmSize",
			"vnetSubnetID": "properties.agentPoolProfiles.0.vnetSubnetID"
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
		"authProfile": {
			"identityProviders": [
				{
					"name": "properties.authProfile.identityProviders.0.name",
					"provider": {
						"kind": "AADIdentityProvider",
						"clientId": "properties.authProfile.identityProviders.0.provider.clientId",
						"secret": "properties.authProfile.identityProviders.0.provider.secret",
						"tenantId": "properties.authProfile.identityProviders.0.provider.tenantId"
					}
				}
			]
		}
	}
}`)

func TestMarshal(t *testing.T) {
	b, err := json.MarshalIndent(unmarshalled, "", "\t")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, marshalled) {
		t.Errorf("json.MarshalIndent returned unexpected result\n%s\n", string(b))
	}
}

func TestUnmarshal(t *testing.T) {
	var oc *OpenShiftManagedCluster
	err := json.Unmarshal(marshalled, &oc)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(oc, unmarshalled) {
		t.Errorf("json.Unmarshal returned unexpected result\n%#v\n", oc)
	}
}

func TestStructTypes(t *testing.T) {
	// AgentPoolProfile and MasterPoolProfile types should be identical bar
	// `Role AgentPoolProfileRole` in the former
	// MasterPoolProfile has removed OsType, Name
	app := reflect.TypeOf(AgentPoolProfile{})
	mpp := reflect.TypeOf(MasterPoolProfile{})
	// Add 3 for Role,OsType,Name which are missing from MasterPoolProfile
	if app.NumField() != mpp.NumField()+3 {
		t.Fatalf("mismatch in number of fields: %d vs %d", mpp.NumField(), app.NumField())
	}
	for i := 0; i < mpp.NumField(); i++ {
		mf := mpp.Field(i)
		af, found := app.FieldByName(mf.Name)
		if !found {
			t.Errorf("field not found in agentpoolprofile: %s", mf.Name)
		}
		if !(mf.Type == af.Type || mf.Tag == af.Tag) {
			t.Errorf("mismatch in field %d:\n%#v\n%#v", i, af, mf)
		}
	}
	if app.Field(app.NumField()-1).Name != "Role" {
		t.Errorf("unexpected field name %s", app.Field(app.NumField()-1).Name)
	}
}
