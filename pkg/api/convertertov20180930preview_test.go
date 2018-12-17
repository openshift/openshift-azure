package api

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
)

// internalManagedCluster and v20180930previewManagedCluster are defined in
// converterfromv20180930preview_test.go.

func TestConvertToV20180930preview(t *testing.T) {
	zeroint := 0
	tests := []struct {
		name string
		cs   *OpenShiftManagedCluster
		oc   *v20180930preview.OpenShiftManagedCluster
	}{
		{
			name: "convert populated",
			cs:   internalManagedCluster(),
			oc:   v20180930previewManagedCluster(),
		},
		{
			name: "convert empty structs",
			cs:   &OpenShiftManagedCluster{},
			oc: &v20180930preview.OpenShiftManagedCluster{
				Plan: &v20180930preview.ResourcePurchasePlan{},
				Properties: &v20180930preview.Properties{
					NetworkProfile:    &v20180930preview.NetworkProfile{},
					RouterProfiles:    []v20180930preview.RouterProfile{},
					AgentPoolProfiles: []v20180930preview.AgentPoolProfile{},
					AuthProfile: &v20180930preview.AuthProfile{
						IdentityProviders: []v20180930preview.IdentityProvider{},
					},
				},
				Tags: map[string]*string{},
			},
		},
		{
			name: "convert structs with empty strings",
			cs: &OpenShiftManagedCluster{
				Plan: ResourcePurchasePlan{
					Name:          "",
					Product:       "",
					PromotionCode: "",
					Publisher:     "",
				},
				Properties: Properties{
					ProvisioningState: ProvisioningState(""),
					OpenShiftVersion:  "",
					PublicHostname:    "",
					FQDN:              "",
					NetworkProfile: NetworkProfile{
						VnetCIDR:   "",
						PeerVnetID: "",
					},
					RouterProfiles: []RouterProfile{
						{
							Name:            "",
							PublicSubdomain: "",
							FQDN:            "",
						},
					},
					AgentPoolProfiles: []AgentPoolProfile{
						{
							Name:       "",
							Count:      0,
							SubnetCIDR: "",
							OSType:     OSType(""),
							Role:       AgentPoolProfileRole(""),
						},
					},
					AuthProfile:             AuthProfile{},
					ServicePrincipalProfile: ServicePrincipalProfile{},
					AzProfile:               AzProfile{},
				},
			},
			oc: &v20180930preview.OpenShiftManagedCluster{
				Plan: &v20180930preview.ResourcePurchasePlan{},
				Properties: &v20180930preview.Properties{
					NetworkProfile: &v20180930preview.NetworkProfile{},
					RouterProfiles: []v20180930preview.RouterProfile{
						{},
					},
					AgentPoolProfiles: []v20180930preview.AgentPoolProfile{
						{
							Count: &zeroint,
						},
					},
					AuthProfile: &v20180930preview.AuthProfile{
						IdentityProviders: []v20180930preview.IdentityProvider{},
					},
				},
				Tags: map[string]*string{},
			},
		},
	}

	for _, test := range tests {
		oc := ConvertToV20180930preview(test.cs)
		if !reflect.DeepEqual(oc, test.oc) {
			t.Errorf("%s - unexpected result:\n%#v\nexpected:\n%#v", test.name, spew.Sprint(oc), spew.Sprint(test.oc))
		}
	}
}
