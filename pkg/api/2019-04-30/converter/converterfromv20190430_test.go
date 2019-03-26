package converter

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-test/deep"

	"github.com/openshift/openshift-azure/pkg/api"
	v20190430 "github.com/openshift/openshift-azure/pkg/api/2019-04-30/api"
	"github.com/openshift/openshift-azure/test/util/populate"
)

func v20190430ManagedCluster() *v20190430.OpenShiftManagedCluster {
	// use populate.Walk to generate a fully populated
	// v20190430.OpenShiftManagedCluster

	prepare := func(v reflect.Value) {
		switch v.Interface().(type) {
		case []v20190430.IdentityProvider:
			// set the Provider to AADIdentityProvider
			v.Set(reflect.ValueOf([]v20190430.IdentityProvider{{Provider: &v20190430.AADIdentityProvider{Kind: to.StringPtr("AADIdentityProvider")}}}))
		}
	}

	omc := v20190430.OpenShiftManagedCluster{}
	populate.Walk(&omc, prepare)

	return &omc
}

func TestConvertFromv20190430(t *testing.T) {
	tests := []struct {
		name           string
		input          *v20190430.OpenShiftManagedCluster
		base           *api.OpenShiftManagedCluster
		expectedChange func(*api.OpenShiftManagedCluster)
		err            error
	}{
		{
			name:  "create",
			input: v20190430ManagedCluster(),
		},
		{
			name: "router profile update",
			input: &v20190430.OpenShiftManagedCluster{
				Properties: &v20190430.Properties{
					RouterProfiles: []v20190430.RouterProfile{
						{
							Name:            to.StringPtr("Properties.RouterProfiles[0].Name"),
							PublicSubdomain: to.StringPtr("NewPublicSubdomain"),
						},
					},
				},
			},
			base: api.GetInternalMockCluster(),
			expectedChange: func(expectedCs *api.OpenShiftManagedCluster) {
				expectedCs.Properties.RouterProfiles[0].PublicSubdomain = "NewPublicSubdomain"
			},
		},
		{
			name: "missing name in router profile update",
			input: &v20190430.OpenShiftManagedCluster{
				Properties: &v20190430.Properties{
					RouterProfiles: []v20190430.RouterProfile{
						{
							PublicSubdomain: to.StringPtr("NewPublicSubdomain"),
						},
					},
				},
			},
			base: api.GetInternalMockCluster(),
			err:  errors.New("invalid router profile - name is missing"),
		},
		{
			name: "new agent pool profile",
			input: &v20190430.OpenShiftManagedCluster{
				Properties: &v20190430.Properties{
					AgentPoolProfiles: []v20190430.AgentPoolProfile{
						{
							Name:       to.StringPtr("NewName"),
							Count:      to.Int64Ptr(2),
							VMSize:     (*v20190430.VMSize)(to.StringPtr("NewVMSize")),
							SubnetCIDR: to.StringPtr("NewSubnetCIDR"),
							OSType:     (*v20190430.OSType)(to.StringPtr("NewOSType")),
							Role:       (*v20190430.AgentPoolProfileRole)(to.StringPtr("NewRole")),
						},
					},
				},
			},
			base: api.GetInternalMockCluster(),
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
			input: &v20190430.OpenShiftManagedCluster{
				Properties: &v20190430.Properties{
					AgentPoolProfiles: []v20190430.AgentPoolProfile{
						{
							Count:      to.Int64Ptr(2),
							VMSize:     (*v20190430.VMSize)(to.StringPtr("NewVMSize")),
							SubnetCIDR: to.StringPtr("NewSubnetCIDR"),
							OSType:     (*v20190430.OSType)(to.StringPtr("NewOSType")),
							Role:       (*v20190430.AgentPoolProfileRole)(to.StringPtr("NewRole")),
						},
					},
				},
			},
			base: api.GetInternalMockCluster(),
			err:  errors.New("invalid agent pool profile - name is missing"),
		},
		{
			name: "auth profile update",
			input: &v20190430.OpenShiftManagedCluster{
				Properties: &v20190430.Properties{
					AuthProfile: &v20190430.AuthProfile{
						IdentityProviders: []v20190430.IdentityProvider{
							{
								Name: to.StringPtr("Properties.AuthProfile.IdentityProviders[0].Name"),
								Provider: &v20190430.AADIdentityProvider{
									Secret: to.StringPtr("NewSecret"),
								},
							},
						},
					},
				},
			},
			base: api.GetInternalMockCluster(),
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
			input: &v20190430.OpenShiftManagedCluster{
				Properties: &v20190430.Properties{
					AuthProfile: &v20190430.AuthProfile{
						IdentityProviders: []v20190430.IdentityProvider{
							{
								Name: to.StringPtr("Properties.AuthProfile.IdentityProviders[0].Name"),
								Provider: &v20190430.AADIdentityProvider{
									Kind: to.StringPtr("Kind"),
								},
							},
						},
					},
				},
			},
			base: api.GetInternalMockCluster(),
			err:  errors.New("cannot update the kind of the identity provider"),
		},
		{
			name: "missing name in auth profile update",
			input: &v20190430.OpenShiftManagedCluster{
				Properties: &v20190430.Properties{
					AuthProfile: &v20190430.AuthProfile{
						IdentityProviders: []v20190430.IdentityProvider{
							{
								Provider: &v20190430.AADIdentityProvider{
									Kind: to.StringPtr("Kind"),
								},
							},
						},
					},
				},
			},
			base: api.GetInternalMockCluster(),
			err:  errors.New("invalid identity provider - name is missing"),
		},
		{
			name: "nil ResourcePurchasPlan update",
			input: &v20190430.OpenShiftManagedCluster{
				Plan: nil,
			},
			base: api.GetInternalMockCluster(),
			expectedChange: func(expectedCs *api.OpenShiftManagedCluster) {
			},
		},
	}

	for _, test := range tests {
		expected := api.GetInternalMockCluster()
		if test.expectedChange != nil {
			test.expectedChange(expected)
		}

		output, err := ConvertFromv20190430(test.input, test.base)
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

func TestRoundTripv20190430(t *testing.T) {
	start := v20190430ManagedCluster()
	internal, err := ConvertFromv20190430(start, nil)
	if err != nil {
		t.Error(err)
	}
	end := ConvertTov20190430(internal)
	if !reflect.DeepEqual(start, end) {
		t.Errorf("unexpected diff %s", deep.Equal(start, end))
	}
}
