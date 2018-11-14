package api

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/openshift/openshift-azure/pkg/util/structtags"
	"github.com/openshift/openshift-azure/test/util/populate"
)

var marshalled = []byte(`{
	"plan": {
		"name": "Plan.Name",
		"product": "Plan.Product",
		"promotionCode": "Plan.PromotionCode",
		"publisher": "Plan.Publisher"
	},
	"properties": {
		"provisioningState": "Properties.ProvisioningState",
		"openShiftVersion": "Properties.OpenShiftVersion",
		"publicHostname": "Properties.PublicHostname",
		"fqdn": "Properties.FQDN",
		"networkProfile": {
			"vnetCidr": "Properties.NetworkProfile.VnetCIDR",
			"peerVnetId": "Properties.NetworkProfile.PeerVnetID"
		},
		"routerProfiles": [
			{
				"name": "Properties.RouterProfiles[0].Name",
				"publicSubdomain": "Properties.RouterProfiles[0].PublicSubdomain",
				"fqdn": "Properties.RouterProfiles[0].FQDN"
			}
		],
		"masterPoolProfile": {
			"count": 1,
			"vmSize": "Properties.MasterPoolProfile.VMSize",
			"subnetCidr": "Properties.MasterPoolProfile.SubnetCIDR"
		},
		"agentPoolProfiles": [
			{
				"name": "Properties.AgentPoolProfiles[0].Name",
				"count": 1,
				"vmSize": "Properties.AgentPoolProfiles[0].VMSize",
				"subnetCidr": "Properties.AgentPoolProfiles[0].SubnetCIDR",
				"osType": "Properties.AgentPoolProfiles[0].OSType",
				"role": "Properties.AgentPoolProfiles[0].Role"
			}
		],
		"authProfile": {
			"identityProviders": [
				{
					"name": "Properties.AuthProfile.IdentityProviders[0].Name",
					"provider": {
						"kind": "AADIdentityProvider",
						"clientId": "Properties.AuthProfile.IdentityProviders[0].Provider.ClientID",
						"secret": "Properties.AuthProfile.IdentityProviders[0].Provider.Secret",
						"tenantId": "Properties.AuthProfile.IdentityProviders[0].Provider.TenantID"
					}
				}
			]
		}
	},
	"id": "ID",
	"name": "Name",
	"type": "Type",
	"location": "Location",
	"tags": {
		"Tags.key": "Tags.val"
	}
}`)

func TestMarshal(t *testing.T) {
	prepare := func(v reflect.Value) {
		switch v.Interface().(type) {
		case []IdentityProvider:
			// set the Provider to AADIdentityProvider
			v.Set(reflect.ValueOf([]IdentityProvider{{Provider: &AADIdentityProvider{Kind: "AADIdentityProvider"}}}))
		}
	}

	populatedOc := OpenShiftManagedCluster{}
	populate.Walk(&populatedOc, prepare)

	b, err := json.MarshalIndent(populatedOc, "", "\t")
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(b, marshalled) {
		t.Errorf("json.MarshalIndent returned unexpected result\n%s\n", string(b))
	}
}

func TestUnmarshal(t *testing.T) {
	prepare := func(v reflect.Value) {
		switch v.Interface().(type) {
		case []IdentityProvider:
			// set the Provider to AADIdentityProvider
			v.Set(reflect.ValueOf([]IdentityProvider{{Provider: &AADIdentityProvider{Kind: "AADIdentityProvider"}}}))
		}
	}

	populatedOc := OpenShiftManagedCluster{}
	populate.Walk(&populatedOc, prepare)

	var unmarshalledOc OpenShiftManagedCluster
	err := json.Unmarshal(marshalled, &unmarshalledOc)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(populatedOc, unmarshalledOc) {
		t.Errorf("json.Unmarshal returned unexpected result\n%#v\n", unmarshalledOc)
	}
}

func TestStructTypes(t *testing.T) {
	populateFields := func(t reflect.Type) map[string]reflect.StructField {
		m := map[string]reflect.StructField{}

		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			f.Index = nil
			f.Offset = 0

			m[f.Name] = f
		}

		return m
	}

	appFields := populateFields(reflect.TypeOf(AgentPoolProfile{}))
	mppFields := populateFields(reflect.TypeOf(MasterPoolProfile{}))

	// every field (except Name, OSType, Role) in AgentPoolProfile should be
	// identical in MasterPoolProfile
	for name := range appFields {
		switch name {
		case "Name", "OSType", "Role":
			continue
		}

		if !reflect.DeepEqual(appFields[name], mppFields[name]) {
			t.Errorf("mismatch in field %s:\n%#v\n%#v", name, appFields[name], mppFields[name])
		}
	}

	// every field in MasterPoolProfile should be identical in
	// AgentPoolProfile
	for name := range mppFields {
		if !reflect.DeepEqual(appFields[name], mppFields[name]) {
			t.Errorf("mismatch in field %s:\n%#v\n%#v", name, appFields[name], mppFields[name])
		}
	}
}

// TestJSONTags ensures that all the `json:"..."` struct field tags under
// OpenShiftManagedCluster correspond with their field names
func TestJSONTags(t *testing.T) {
	o := OpenShiftManagedCluster{}
	for _, err := range structtags.CheckJsonTags(o) {
		t.Errorf("mismatch in struct tags for %T: %s", o, err.Error())
	}
}
