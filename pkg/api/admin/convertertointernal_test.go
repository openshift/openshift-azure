package admin

import (
	"crypto/x509"
	"errors"
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-test/deep"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/tls"
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

func internalManagedCluster() *api.OpenShiftManagedCluster {
	cs := api.GetInternalMockCluster(false)

	prepare := func(v reflect.Value) {
		switch v.Interface().(type) {
		case []api.IdentityProvider:
			// set the Provider to AADIdentityProvider
			v.Set(reflect.ValueOf([]api.IdentityProvider{{Provider: &api.AADIdentityProvider{Kind: "AADIdentityProvider"}}}))
		}
	}
	populate.Walk(&cs, prepare)
	return cs
}

func TestToInternal(t *testing.T) {
	_, dummyCA, err := tls.NewCA("dummy")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		input          *OpenShiftManagedCluster
		base           *api.OpenShiftManagedCluster
		expectedChange func(*api.OpenShiftManagedCluster)
		err            error
	}{
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
			base: internalManagedCluster(),
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
			base: internalManagedCluster(),
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
			base: internalManagedCluster(),
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
			base: internalManagedCluster(),
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
									ClientID: to.StringPtr("NewClientID"),
								},
							},
						},
					},
				},
			},
			base: internalManagedCluster(),
			expectedChange: func(expectedCs *api.OpenShiftManagedCluster) {
				expectedCs.Properties.AuthProfile = api.AuthProfile{
					IdentityProviders: []api.IdentityProvider{
						{
							Name: "Properties.AuthProfile.IdentityProviders[0].Name",
							Provider: &api.AADIdentityProvider{
								Kind:     "AADIdentityProvider",
								ClientID: "NewClientID",
								Secret:   "Properties.AuthProfile.IdentityProviders[0].Provider.Secret",
								TenantID: "Properties.AuthProfile.IdentityProviders[0].Provider.TenantID",
							},
						},
					},
				}
			},
		},
		{
			name: "auth profile update aad groups",
			input: &OpenShiftManagedCluster{
				Properties: &Properties{
					AuthProfile: &AuthProfile{
						IdentityProviders: []IdentityProvider{
							{
								Name: to.StringPtr("Properties.AuthProfile.IdentityProviders[0].Name"),
								Provider: &AADIdentityProvider{
									CustomerAdminGroupID: to.StringPtr("admin"),
								},
							},
						},
					},
				},
			},
			base: internalManagedCluster(),
			expectedChange: func(expectedCs *api.OpenShiftManagedCluster) {
				expectedCs.Properties.AuthProfile = api.AuthProfile{
					IdentityProviders: []api.IdentityProvider{
						{
							Name: "Properties.AuthProfile.IdentityProviders[0].Name",
							Provider: &api.AADIdentityProvider{
								Kind:                 "AADIdentityProvider",
								ClientID:             "Properties.AuthProfile.IdentityProviders[0].Provider.ClientID",
								Secret:               "Properties.AuthProfile.IdentityProviders[0].Provider.Secret",
								TenantID:             "Properties.AuthProfile.IdentityProviders[0].Provider.TenantID",
								CustomerAdminGroupID: to.StringPtr("admin"),
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
			base: internalManagedCluster(),
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
			base: internalManagedCluster(),
			err:  errors.New("invalid identity provider - name is missing"),
		},
		{
			name: "image offer update",
			input: &OpenShiftManagedCluster{
				Config: &Config{
					ImageOffer: to.StringPtr("NewOffer"),
				},
			},
			base: internalManagedCluster(),
			expectedChange: func(expectedCs *api.OpenShiftManagedCluster) {
				expectedCs.Config.ImageOffer = "NewOffer"
			},
		},
		{
			name: "certificate update",
			input: &OpenShiftManagedCluster{
				Config: &Config{
					Certificates: &CertificateConfig{
						OpenShiftConsole: &CertificateChain{
							Certs: []*x509.Certificate{dummyCA},
						},
					},
				},
			},
			base: internalManagedCluster(),
			expectedChange: func(expectedCs *api.OpenShiftManagedCluster) {
				expectedCs.Config.Certificates.OpenShiftConsole.Certs = []*x509.Certificate{dummyCA}
			},
		},
		{
			name: "loglevel update",
			input: &OpenShiftManagedCluster{
				Config: &Config{
					ComponentLogLevel: &ComponentLogLevel{
						Node: to.IntPtr(2),
					},
				},
			},
			base: internalManagedCluster(),
			expectedChange: func(expectedCs *api.OpenShiftManagedCluster) {
				expectedCs.Config.ComponentLogLevel.Node = to.IntPtr(2)
			},
		},
	}

	for _, test := range tests {
		expected := internalManagedCluster()
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
