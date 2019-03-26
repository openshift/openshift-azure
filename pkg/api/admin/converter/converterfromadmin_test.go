package api

import (
	"crypto/x509"
	"errors"
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-test/deep"

	"github.com/openshift/openshift-azure/pkg/api"
	admin "github.com/openshift/openshift-azure/pkg/api/admin/api"
	"github.com/openshift/openshift-azure/pkg/util/tls"
	"github.com/openshift/openshift-azure/test/util/populate"
)

func adminManagedCluster() *admin.OpenShiftManagedCluster {
	// use populate.Walk to generate a fully populated
	// admin.OpenShiftManagedCluster

	prepare := func(v reflect.Value) {
		switch v.Interface().(type) {
		case []admin.IdentityProvider:
			// set the Provider to AADIdentityProvider
			v.Set(reflect.ValueOf([]admin.IdentityProvider{{Provider: &admin.AADIdentityProvider{Kind: to.StringPtr("AADIdentityProvider")}}}))
		}
	}

	omc := admin.OpenShiftManagedCluster{}
	populate.Walk(&omc, prepare)

	return &omc
}

func internalManagedCluster() *api.OpenShiftManagedCluster {
	// this is the expected internal equivalent to
	// v20180930previewManagedCluster()

	return &api.OpenShiftManagedCluster{
		ID:       "ID",
		Location: "Location",
		Name:     "Name",
		Plan: &api.ResourcePurchasePlan{
			Name:          to.StringPtr("Plan.Name"),
			Product:       to.StringPtr("Plan.Product"),
			PromotionCode: to.StringPtr("Plan.PromotionCode"),
			Publisher:     to.StringPtr("Plan.Publisher"),
		},
		Tags: map[string]string{
			"Tags.key": "Tags.val",
		},
		Type: "Type",
		Properties: api.Properties{
			ProvisioningState: "Properties.ProvisioningState",
			OpenShiftVersion:  "Properties.OpenShiftVersion",
			ClusterVersion:    "Properties.ClusterVersion",
			PublicHostname:    "Properties.PublicHostname",
			RouterProfiles: []api.RouterProfile{
				{
					Name:            "Properties.RouterProfiles[0].Name",
					PublicSubdomain: "Properties.RouterProfiles[0].PublicSubdomain",
					FQDN:            "Properties.RouterProfiles[0].FQDN",
				},
			},
			FQDN: "Properties.FQDN",
			AuthProfile: api.AuthProfile{
				IdentityProviders: []api.IdentityProvider{
					{
						Name: "Properties.AuthProfile.IdentityProviders[0].Name",
						Provider: &api.AADIdentityProvider{
							Kind:                 "AADIdentityProvider",
							ClientID:             "Properties.AuthProfile.IdentityProviders[0].Provider.ClientID",
							Secret:               "Properties.AuthProfile.IdentityProviders[0].Provider.Secret",
							TenantID:             "Properties.AuthProfile.IdentityProviders[0].Provider.TenantID",
							CustomerAdminGroupID: to.StringPtr("Properties.AuthProfile.IdentityProviders[0].Provider.CustomerAdminGroupID"),
						},
					},
				},
			},
			NetworkProfile: api.NetworkProfile{
				VnetID:     "Properties.NetworkProfile.VnetID",
				VnetCIDR:   "Properties.NetworkProfile.VnetCIDR",
				PeerVnetID: to.StringPtr("Properties.NetworkProfile.PeerVnetID"),
			},
			AgentPoolProfiles: []api.AgentPoolProfile{
				{
					Name:       string(api.AgentPoolProfileRoleMaster),
					Count:      1,
					VMSize:     "Properties.MasterPoolProfile.VMSize",
					SubnetCIDR: "Properties.MasterPoolProfile.SubnetCIDR",
					OSType:     api.OSTypeLinux,
					Role:       api.AgentPoolProfileRoleMaster,
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

func internalManagedClusterAdmin() *api.OpenShiftManagedCluster {
	cs := internalManagedCluster()

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

func TestConvertFromAdmin(t *testing.T) {
	_, dummyCA, err := tls.NewCA("dummy")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		input          *admin.OpenShiftManagedCluster
		base           *api.OpenShiftManagedCluster
		expectedChange func(*api.OpenShiftManagedCluster)
		err            error
	}{
		{
			name: "router profile update",
			input: &admin.OpenShiftManagedCluster{
				Properties: &admin.Properties{
					RouterProfiles: []admin.RouterProfile{
						{
							Name:            to.StringPtr("Properties.RouterProfiles[0].Name"),
							PublicSubdomain: to.StringPtr("NewPublicSubdomain"),
						},
					},
				},
			},
			base: internalManagedClusterAdmin(),
			expectedChange: func(expectedCs *api.OpenShiftManagedCluster) {
				expectedCs.Properties.RouterProfiles[0].PublicSubdomain = "NewPublicSubdomain"
			},
		},
		{
			name: "missing name in router profile update",
			input: &admin.OpenShiftManagedCluster{
				Properties: &admin.Properties{
					RouterProfiles: []admin.RouterProfile{
						{
							PublicSubdomain: to.StringPtr("NewPublicSubdomain"),
						},
					},
				},
			},
			base: internalManagedClusterAdmin(),
			err:  errors.New("invalid router profile - name is missing"),
		},
		{
			name: "new agent pool profile",
			input: &admin.OpenShiftManagedCluster{
				Properties: &admin.Properties{
					AgentPoolProfiles: []admin.AgentPoolProfile{
						{
							Name:       to.StringPtr("NewName"),
							Count:      to.Int64Ptr(2),
							VMSize:     (*admin.VMSize)(to.StringPtr("NewVMSize")),
							SubnetCIDR: to.StringPtr("NewSubnetCIDR"),
							OSType:     (*admin.OSType)(to.StringPtr("NewOSType")),
							Role:       (*admin.AgentPoolProfileRole)(to.StringPtr("NewRole")),
						},
					},
				},
			},
			base: internalManagedClusterAdmin(),
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
			input: &admin.OpenShiftManagedCluster{
				Properties: &admin.Properties{
					AgentPoolProfiles: []admin.AgentPoolProfile{
						{
							Count:      to.Int64Ptr(2),
							VMSize:     (*admin.VMSize)(to.StringPtr("NewVMSize")),
							SubnetCIDR: to.StringPtr("NewSubnetCIDR"),
							OSType:     (*admin.OSType)(to.StringPtr("NewOSType")),
							Role:       (*admin.AgentPoolProfileRole)(to.StringPtr("NewRole")),
						},
					},
				},
			},
			base: internalManagedClusterAdmin(),
			err:  errors.New("invalid agent pool profile - name is missing"),
		},
		{
			name: "auth profile update",
			input: &admin.OpenShiftManagedCluster{
				Properties: &admin.Properties{
					AuthProfile: &admin.AuthProfile{
						IdentityProviders: []admin.IdentityProvider{
							{
								Name: to.StringPtr("Properties.AuthProfile.IdentityProviders[0].Name"),
								Provider: &admin.AADIdentityProvider{
									ClientID: to.StringPtr("NewClientID"),
								},
							},
						},
					},
				},
			},
			base: internalManagedClusterAdmin(),
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
			input: &admin.OpenShiftManagedCluster{
				Properties: &admin.Properties{
					AuthProfile: &admin.AuthProfile{
						IdentityProviders: []admin.IdentityProvider{
							{
								Name: to.StringPtr("Properties.AuthProfile.IdentityProviders[0].Name"),
								Provider: &admin.AADIdentityProvider{
									CustomerAdminGroupID: to.StringPtr("admin"),
								},
							},
						},
					},
				},
			},
			base: internalManagedClusterAdmin(),
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
			input: &admin.OpenShiftManagedCluster{
				Properties: &admin.Properties{
					AuthProfile: &admin.AuthProfile{
						IdentityProviders: []admin.IdentityProvider{
							{
								Name: to.StringPtr("Properties.AuthProfile.IdentityProviders[0].Name"),
								Provider: &admin.AADIdentityProvider{
									Kind: to.StringPtr("Kind"),
								},
							},
						},
					},
				},
			},
			base: internalManagedClusterAdmin(),
			err:  errors.New("cannot update the kind of the identity provider"),
		},
		{
			name: "missing name in auth profile update",
			input: &admin.OpenShiftManagedCluster{
				Properties: &admin.Properties{
					AuthProfile: &admin.AuthProfile{
						IdentityProviders: []admin.IdentityProvider{
							{
								Provider: &admin.AADIdentityProvider{
									Kind: to.StringPtr("Kind"),
								},
							},
						},
					},
				},
			},
			base: internalManagedClusterAdmin(),
			err:  errors.New("invalid identity provider - name is missing"),
		},
		{
			name: "image offer update",
			input: &admin.OpenShiftManagedCluster{
				Config: &admin.Config{
					ImageOffer: to.StringPtr("NewOffer"),
				},
			},
			base: internalManagedClusterAdmin(),
			expectedChange: func(expectedCs *api.OpenShiftManagedCluster) {
				expectedCs.Config.ImageOffer = "NewOffer"
			},
		},
		{
			name: "certificate update",
			input: &admin.OpenShiftManagedCluster{
				Config: &admin.Config{
					Certificates: &admin.CertificateConfig{
						OpenShiftConsole: &admin.CertificateChain{
							Certs: []*x509.Certificate{dummyCA},
						},
					},
				},
			},
			base: internalManagedClusterAdmin(),
			expectedChange: func(expectedCs *api.OpenShiftManagedCluster) {
				expectedCs.Config.Certificates.OpenShiftConsole.Certs = []*x509.Certificate{dummyCA}
			},
		},
		{
			name: "loglevel update",
			input: &admin.OpenShiftManagedCluster{
				Config: &admin.Config{
					ComponentLogLevel: &admin.ComponentLogLevel{
						Node: to.IntPtr(2),
					},
				},
			},
			base: internalManagedClusterAdmin(),
			expectedChange: func(expectedCs *api.OpenShiftManagedCluster) {
				expectedCs.Config.ComponentLogLevel.Node = to.IntPtr(2)
			},
		},
	}

	for _, test := range tests {
		expected := internalManagedClusterAdmin()
		if test.expectedChange != nil {
			test.expectedChange(expected)
		}

		output, err := ConvertFromAdmin(test.input, test.base)
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

func TestRoundTripAdmin(t *testing.T) {
	start := adminManagedCluster()
	internal, err := ConvertFromAdmin(start, nil)
	if err != nil {
		t.Error(err)
	}
	end := ConvertToAdmin(internal)
	if !reflect.DeepEqual(start, end) {
		t.Errorf("unexpected diff %s", deep.Equal(start, end))
	}
}
