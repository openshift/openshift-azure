package cluster

import (
	"context"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/tls"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_azureclient"
	"github.com/openshift/openshift-azure/test/util/populate"
)

func TestHashScaleSet(t *testing.T) {
	tests := []struct {
		name string
		app  api.AgentPoolProfile
	}{
		{
			name: "hash shouldn't change over time",
		},
		{
			name: "hash is invariant with name and count",
			app: api.AgentPoolProfile{
				Name:  "foo",
				Count: 1,
			},
		},
	}

	prepare := func(v reflect.Value) {
		switch v.Interface().(type) {
		case []api.IdentityProvider:
			// set the Provider to AADIdentityProvider
			v.Set(reflect.ValueOf([]api.IdentityProvider{{Provider: &api.AADIdentityProvider{Kind: "AADIdentityProvider"}}}))
		}
	}

	_, cert1, err := tls.NewCA("dummy-test-certificate1.local")
	if err != nil {
		t.Errorf("NewCa : unexpected error: %v", err)
	}
	_, cert2, err := tls.NewCA("dummy-test-certificate2.local")
	if err != nil {
		t.Errorf("NewCa : unexpected error: %v", err)
	}
	var cs api.OpenShiftManagedCluster
	populate.Walk(&cs, prepare)
	cs.Properties.APICertProfile.KeyVaultSecretURL = "https://unittest.vault.azure.net/secrets/PublicHostname"
	cs.Properties.RouterProfiles[0].RouterCertProfile.KeyVaultSecretURL = "https://unittest.vault.azure.net/secrets/Router"

	var h hasher
	var exp []byte
	for _, test := range tests {
		gmc := gomock.NewController(t)
		defer gmc.Finish()
		ctx := context.Background()
		kvc := mock_azureclient.NewMockKeyVaultClient(gmc)

		certBytes1, err := tls.CertAsBytes(cert1)
		if err != nil {
			t.Errorf("%s:CertAsBytes unexpected error: %v", test.name, err)
		}
		certBytes2, err := tls.CertAsBytes(cert2)
		if err != nil {
			t.Errorf("%s:CertAsBytes unexpected error: %v", test.name, err)
		}

		secret1 := keyvault.SecretBundle{Value: to.StringPtr(string(certBytes1))}
		secret2 := keyvault.SecretBundle{Value: to.StringPtr(string(certBytes2))}

		kvc.EXPECT().GetSecret(ctx, "https://unittest.vault.azure.net", "PublicHostname", "").Return(secret1, nil)
		kvc.EXPECT().GetSecret(ctx, "https://unittest.vault.azure.net", "Router", "").Return(secret2, nil)
		h.kvc = kvc

		got, err := h.HashScaleSet(&cs, &test.app)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", test.name, err)
		}
		if exp == nil {
			exp = got
		}
		if !reflect.DeepEqual(got, exp) {
			t.Errorf("%s: expected:\n%#v\ngot:\n%#v", test.name, exp, got)
		}
	}
}
