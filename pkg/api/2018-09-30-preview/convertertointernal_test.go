package v20180930preview

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-test/deep"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/test/util/populate"
)

func managedCluster() *OpenShiftManagedCluster {
	// use populate.Walk to generate a fully populated
	// OpenShiftManagedCluster
	prepare := func(v reflect.Value) {
		switch v.Interface().(type) {
		case []IdentityProvider:
			// set the Provider to AADIdentityProvider
			v.Set(reflect.ValueOf([]IdentityProvider{{Provider: &AADIdentityProvider{Kind: to.StringPtr("AADIdentityProvider")}}}))
		}
	}

	omc := OpenShiftManagedCluster{}
	populate.Walk(&omc, prepare)

	return &omc
}

func TestToInternal(t *testing.T) {
	provisioningState := ProvisioningState(Creating)
	tests := []struct {
		name           string
		input          *OpenShiftManagedCluster
		base           *api.OpenShiftManagedCluster
		expectedChange func(*api.OpenShiftManagedCluster)
		err            error
	}{
		{
			name:  "create",
			input: managedCluster(),
		},
		{
			name: "router profile update",
			input: &OpenShiftManagedCluster{
				Properties: &Properties{
					RouterProfiles: []RouterProfile{
						{
							Name:            to.StringPtr("Properties.RouterProfiles[0].Name"),
							PublicSubdomain: to.StringPtr("NewPublicSubdomain"),
						},
					},
				},
			},
			base: api.GetInternalMockCluster(false),
			expectedChange: func(expectedCs *api.OpenShiftManagedCluster) {
				expectedCs.Properties.RouterProfiles[0].PublicSubdomain = "NewPublicSubdomain"
			},
		},
		{
			name: "missing name in router profile update",
			input: &OpenShiftManagedCluster{
				Properties: &Properties{
					RouterProfiles: []RouterProfile{
						{
							PublicSubdomain: to.StringPtr("NewPublicSubdomain"),
						},
					},
				},
			},
			base: api.GetInternalMockCluster(false),
			err:  errors.New("invalid router profile - name is missing"),
		},
		{
			name: "new agent pool profile",
			input: &OpenShiftManagedCluster{
				Properties: &Properties{
					AgentPoolProfiles: []AgentPoolProfile{
						{
							Name:       to.StringPtr("NewName"),
							Count:      to.Int64Ptr(2),
							VMSize:     (*VMSize)(to.StringPtr("NewVMSize")),
							SubnetCIDR: to.StringPtr("NewSubnetCIDR"),
							OSType:     (*OSType)(to.StringPtr("NewOSType")),
							Role:       (*AgentPoolProfileRole)(to.StringPtr("NewRole")),
						},
					},
				},
			},
			base: api.GetInternalMockCluster(false),
			expectedChange: func(expectedCs *api.OpenShiftManagedCluster) {
				expectedCs.Properties.AgentPoolProfiles = append(expectedCs.Properties.AgentPoolProfiles,
					api.AgentPoolProfile{
						Name:       "NewName",
						Count:      2,
						VMSize:     api.VMSize("NewVMSize"),
						SubnetCIDR: "NewSubnetCIDR",
						OSType:     api.OSType("NewOSType"),
						Role:       api.AgentPoolProfileRole("NewRole"),
					})
			},
		},
		{
			name: "missing name in agent pool profile update",
			input: &OpenShiftManagedCluster{
				Properties: &Properties{
					AgentPoolProfiles: []AgentPoolProfile{
						{
							Count:      to.Int64Ptr(2),
							VMSize:     (*VMSize)(to.StringPtr("NewVMSize")),
							SubnetCIDR: to.StringPtr("NewSubnetCIDR"),
							OSType:     (*OSType)(to.StringPtr("NewOSType")),
							Role:       (*AgentPoolProfileRole)(to.StringPtr("NewRole")),
						},
					},
				},
			},
			base: api.GetInternalMockCluster(false),
			err:  errors.New("invalid agent pool profile - name is missing"),
		},
		{
			name: "auth profile update",
			input: &OpenShiftManagedCluster{
				Properties: &Properties{
					AuthProfile: &AuthProfile{
						IdentityProviders: []IdentityProvider{
							{
								Name: to.StringPtr("Properties.AuthProfile.IdentityProviders[0].Name"),
								Provider: &AADIdentityProvider{
									Secret: to.StringPtr("NewSecret"),
								},
							},
						},
					},
				},
			},
			base: api.GetInternalMockCluster(false),
			expectedChange: func(expectedCs *api.OpenShiftManagedCluster) {
				expectedCs.Properties.AuthProfile = api.AuthProfile{
					IdentityProviders: []api.IdentityProvider{
						{
							Name: "Properties.AuthProfile.IdentityProviders[0].Name",
							Provider: &api.AADIdentityProvider{
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
			input: &OpenShiftManagedCluster{
				Properties: &Properties{
					AuthProfile: &AuthProfile{
						IdentityProviders: []IdentityProvider{
							{
								Name: to.StringPtr("Properties.AuthProfile.IdentityProviders[0].Name"),
								Provider: &AADIdentityProvider{
									Kind: to.StringPtr("Kind"),
								},
							},
						},
					},
				},
			},
			base: api.GetInternalMockCluster(false),
			err:  errors.New("cannot update the kind of the identity provider"),
		},
		{
			name: "missing name in auth profile update",
			input: &OpenShiftManagedCluster{
				Properties: &Properties{
					AuthProfile: &AuthProfile{
						IdentityProviders: []IdentityProvider{
							{
								Provider: &AADIdentityProvider{
									Kind: to.StringPtr("Kind"),
								},
							},
						},
					},
				},
			},
			base: api.GetInternalMockCluster(false),
			err:  errors.New("invalid identity provider - name is missing"),
		},
		{
			name: "nil ResourcePurchasPlan update",
			input: &OpenShiftManagedCluster{
				Plan: nil,
			},
			base: api.GetInternalMockCluster(false),
			expectedChange: func(expectedCs *api.OpenShiftManagedCluster) {
			},
		},
		{
			name: "dropped ProvisioningState",
			input: &OpenShiftManagedCluster{
				Properties: &Properties{
					ProvisioningState: &provisioningState,
				},
			},
			base: api.GetInternalMockCluster(false),
			expectedChange: func(expectedCs *api.OpenShiftManagedCluster) {
			},
		},
	}

	for _, test := range tests {
		expected := api.GetInternalMockCluster(false)
		if test.expectedChange != nil {
			test.expectedChange(expected)
		}

		output, err := ToInternal(test.input, test.base)

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

func TestRoundTrip(t *testing.T) {
	start := managedCluster()
	internal, err := ToInternal(start, nil)
	if err != nil {
		t.Error(err)
	}

	// dropped fields might be in the start,
	// but they will be dropped in the final result
	*start.Properties.ProvisioningState = ""

	end := FromInternal(internal)
	if !reflect.DeepEqual(start, end) {
		t.Errorf("unexpected diff %s", deep.Equal(start, end))
	}
}
