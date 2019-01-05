package api

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-test/deep"

	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	"github.com/openshift/openshift-azure/test/util/populate"
)

func v20180930previewManagedCluster() *v20180930preview.OpenShiftManagedCluster {
	// use populate.Walk to generate a fully populated
	// v20180930preview.OpenShiftManagedCluster

	prepare := func(v reflect.Value) {
		switch v.Interface().(type) {
		case []v20180930preview.IdentityProvider:
			// set the Provider to AADIdentityProvider
			v.Set(reflect.ValueOf([]v20180930preview.IdentityProvider{{Provider: &v20180930preview.AADIdentityProvider{Kind: to.StringPtr("AADIdentityProvider")}}}))
		}
	}

	omc := v20180930preview.OpenShiftManagedCluster{}
	populate.Walk(&omc, prepare)

	return &omc
}

func internalManagedCluster() *OpenShiftManagedCluster {
	// this is the expected internal equivalent to
	// v20180930previewManagedCluster()

	return &OpenShiftManagedCluster{
		ID:       "ID",
		Location: "Location",
		Name:     "Name",
		Plan: &ResourcePurchasePlan{
			Name:          to.StringPtr("Plan.Name"),
			Product:       to.StringPtr("Plan.Product"),
			PromotionCode: to.StringPtr("Plan.PromotionCode"),
			Publisher:     to.StringPtr("Plan.Publisher"),
		},
		Tags: map[string]string{
			"Tags.key": "Tags.val",
		},
		Type: "Type",
		Properties: Properties{
			ProvisioningState: "Properties.ProvisioningState",
			OpenShiftVersion:  "Properties.OpenShiftVersion",
			PublicHostname:    "Properties.PublicHostname",
			RouterProfiles: []RouterProfile{
				{
					Name:            "Properties.RouterProfiles[0].Name",
					PublicSubdomain: "Properties.RouterProfiles[0].PublicSubdomain",
					FQDN:            "Properties.RouterProfiles[0].FQDN",
				},
			},
			FQDN: "Properties.FQDN",
			AuthProfile: AuthProfile{
				IdentityProviders: []IdentityProvider{
					{
						Name: "Properties.AuthProfile.IdentityProviders[0].Name",
						Provider: &AADIdentityProvider{
							Kind:     "AADIdentityProvider",
							ClientID: "Properties.AuthProfile.IdentityProviders[0].Provider.ClientID",
							Secret:   "Properties.AuthProfile.IdentityProviders[0].Provider.Secret",
							TenantID: "Properties.AuthProfile.IdentityProviders[0].Provider.TenantID",
						},
					},
				},
			},
			NetworkProfile: NetworkProfile{
				VnetCIDR:   "Properties.NetworkProfile.VnetCIDR",
				PeerVnetID: "Properties.NetworkProfile.PeerVnetID",
			},
			AgentPoolProfiles: []AgentPoolProfile{
				{
					Name:       string(AgentPoolProfileRoleMaster),
					Count:      1,
					VMSize:     "Properties.MasterPoolProfile.VMSize",
					SubnetCIDR: "Properties.MasterPoolProfile.SubnetCIDR",
					OSType:     OSTypeLinux,
					Role:       AgentPoolProfileRoleMaster,
				},
				{
					Name:       "Properties.AgentPoolProfiles[0].Name",
					Count:      1,
					VMSize:     "Properties.AgentPoolProfiles[0].VMSize",
					SubnetCIDR: "Properties.AgentPoolProfiles[0].SubnetCIDR",
					OSType:     "Properties.AgentPoolProfiles[0].OSType",
					Role:       "Properties.AgentPoolProfiles[0].Role",
				},
			},
		},
	}
}

func TestConvertFromV20180930preview(t *testing.T) {
	tests := []struct {
		name           string
		input          *v20180930preview.OpenShiftManagedCluster
		base           *OpenShiftManagedCluster
		expectedChange func(*OpenShiftManagedCluster)
		err            error
	}{
		{
			name:  "create",
			input: v20180930previewManagedCluster(),
		},
		{
			name: "router profile update",
			input: &v20180930preview.OpenShiftManagedCluster{
				Properties: &v20180930preview.Properties{
					RouterProfiles: []v20180930preview.RouterProfile{
						{
							Name:            to.StringPtr("Properties.RouterProfiles[0].Name"),
							PublicSubdomain: to.StringPtr("NewPublicSubdomain"),
						},
					},
				},
			},
			base: internalManagedCluster(),
			expectedChange: func(expectedCs *OpenShiftManagedCluster) {
				expectedCs.Properties.RouterProfiles[0].PublicSubdomain = "NewPublicSubdomain"
			},
		},
		{
			name: "missing name in router profile update",
			input: &v20180930preview.OpenShiftManagedCluster{
				Properties: &v20180930preview.Properties{
					RouterProfiles: []v20180930preview.RouterProfile{
						{
							PublicSubdomain: to.StringPtr("NewPublicSubdomain"),
						},
					},
				},
			},
			base: internalManagedCluster(),
			err:  errors.New("invalid router profile - name is missing"),
		},
		{
			name: "new agent pool profile",
			input: &v20180930preview.OpenShiftManagedCluster{
				Properties: &v20180930preview.Properties{
					AgentPoolProfiles: []v20180930preview.AgentPoolProfile{
						{
							Name:       to.StringPtr("NewName"),
							Count:      to.Int64Ptr(2),
							VMSize:     (*v20180930preview.VMSize)(to.StringPtr("NewVMSize")),
							SubnetCIDR: to.StringPtr("NewSubnetCIDR"),
							OSType:     (*v20180930preview.OSType)(to.StringPtr("NewOSType")),
							Role:       (*v20180930preview.AgentPoolProfileRole)(to.StringPtr("NewRole")),
						},
					},
				},
			},
			base: internalManagedCluster(),
			expectedChange: func(expectedCs *OpenShiftManagedCluster) {
				expectedCs.Properties.AgentPoolProfiles = append(expectedCs.Properties.AgentPoolProfiles,
					AgentPoolProfile{
						Name:       "NewName",
						Count:      2,
						VMSize:     VMSize("NewVMSize"),
						SubnetCIDR: "NewSubnetCIDR",
						OSType:     OSType("NewOSType"),
						Role:       AgentPoolProfileRole("NewRole"),
					})
			},
		},
		{
			name: "missing name in agent pool profile update",
			input: &v20180930preview.OpenShiftManagedCluster{
				Properties: &v20180930preview.Properties{
					AgentPoolProfiles: []v20180930preview.AgentPoolProfile{
						{
							Count:      to.Int64Ptr(2),
							VMSize:     (*v20180930preview.VMSize)(to.StringPtr("NewVMSize")),
							SubnetCIDR: to.StringPtr("NewSubnetCIDR"),
							OSType:     (*v20180930preview.OSType)(to.StringPtr("NewOSType")),
							Role:       (*v20180930preview.AgentPoolProfileRole)(to.StringPtr("NewRole")),
						},
					},
				},
			},
			base: internalManagedCluster(),
			err:  errors.New("invalid agent pool profile - name is missing"),
		},
		{
			name: "auth profile update",
			input: &v20180930preview.OpenShiftManagedCluster{
				Properties: &v20180930preview.Properties{
					AuthProfile: &v20180930preview.AuthProfile{
						IdentityProviders: []v20180930preview.IdentityProvider{
							{
								Name: to.StringPtr("Properties.AuthProfile.IdentityProviders[0].Name"),
								Provider: &v20180930preview.AADIdentityProvider{
									Secret: to.StringPtr("NewSecret"),
								},
							},
						},
					},
				},
			},
			base: internalManagedCluster(),
			expectedChange: func(expectedCs *OpenShiftManagedCluster) {
				expectedCs.Properties.AuthProfile = AuthProfile{
					IdentityProviders: []IdentityProvider{
						{
							Name: "Properties.AuthProfile.IdentityProviders[0].Name",
							Provider: &AADIdentityProvider{
								Kind:     "AADIdentityProvider",
								ClientID: "Properties.AuthProfile.IdentityProviders[0].Provider.ClientID",
								Secret:   "NewSecret",
								TenantID: "Properties.AuthProfile.IdentityProviders[0].Provider.TenantID",
							},
						},
					},
				}
			},
		},
		{
			name: "invalid auth profile update",
			input: &v20180930preview.OpenShiftManagedCluster{
				Properties: &v20180930preview.Properties{
					AuthProfile: &v20180930preview.AuthProfile{
						IdentityProviders: []v20180930preview.IdentityProvider{
							{
								Name: to.StringPtr("Properties.AuthProfile.IdentityProviders[0].Name"),
								Provider: &v20180930preview.AADIdentityProvider{
									Kind: to.StringPtr("HtPasswd"),
								},
							},
						},
					},
				},
			},
			base: internalManagedCluster(),
			err:  errors.New("cannot update the kind of the identity provider"),
		},
		{
			name: "missing name in auth profile update",
			input: &v20180930preview.OpenShiftManagedCluster{
				Properties: &v20180930preview.Properties{
					AuthProfile: &v20180930preview.AuthProfile{
						IdentityProviders: []v20180930preview.IdentityProvider{
							{
								Provider: &v20180930preview.AADIdentityProvider{
									Kind: to.StringPtr("HtPasswd"),
								},
							},
						},
					},
				},
			},
			base: internalManagedCluster(),
			err:  errors.New("invalid identity provider - name is missing"),
		},
		{
			name: "nil ResourcePurchasPlan update",
			input: &v20180930preview.OpenShiftManagedCluster{
				Plan: nil,
			},
			base: internalManagedCluster(),
			expectedChange: func(expectedCs *OpenShiftManagedCluster) {
			},
		},
	}

	for _, test := range tests {
		expected := internalManagedCluster()
		if test.expectedChange != nil {
			test.expectedChange(expected)
		}

		output, err := ConvertFromV20180930preview(test.input, test.base)
		if !reflect.DeepEqual(err, test.err) {
			t.Errorf("%s: expected error: %v, got error: %v", test.name, test.err, err)
		}
		if err == nil {
			if !reflect.DeepEqual(output, expected) {
				t.Errorf("%s: unexpected diff %s", test.name, deep.Equal(output, expected))
			}
		}
	}
}
