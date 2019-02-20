package cluster

//go:generate go get github.com/golang/mock/gomock
//go:generate go install github.com/golang/mock/mockgen
//go:generate mockgen -destination=../util/mocks/mock_$GOPACKAGE/hash.go -package=mock_$GOPACKAGE -source hash.go
//go:generate gofmt -s -l -w ../util/mocks/mock_$GOPACKAGE/hash.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../util/mocks/mock_$GOPACKAGE/hash.go

import (
	"context"
	"crypto/sha256"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/tls"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

type Hasher interface {
	HashScaleSet(*api.OpenShiftManagedCluster, *api.AgentPoolProfile) ([]byte, error)
}

type hasher struct {
	pluginConfig api.PluginConfig
	kvc          azureclient.KeyVaultClient
}

func (h *hasher) getCertificateFromVault(ctx context.Context, cs *api.OpenShiftManagedCluster, kvURL string) (string, error) {
	vaultURL, certName, err := azureclient.GetURLCertNameFromFullURL(kvURL)
	if err != nil {
		return "", err
	}
	if h.kvc == nil {
		var err error
		cfg := auth.NewClientCredentialsConfig(cs.Properties.MasterServicePrincipalProfile.ClientID, cs.Properties.MasterServicePrincipalProfile.Secret, cs.Properties.AzProfile.TenantID)
		h.kvc, err = azureclient.NewKeyVaultClient(cfg, vaultURL)
		if err != nil {
			return "", err
		}
	}
	bundle, err := h.kvc.GetSecret(context.Background(), vaultURL, certName, "")
	if err != nil {
		return "", err
	}
	return tls.GetPemBlock([]byte(*bundle.Value), "CERTIFICATE")
}

func hashVMSS(vmss *compute.VirtualMachineScaleSet, cert1, cert2 string) ([]byte, error) {
	data, err := json.Marshal(vmss)
	if err != nil {
		return nil, err
	}
	data = append(data, []byte(cert1)...)
	data = append(data, []byte(cert2)...)

	hf := sha256.New()
	hf.Write(data)

	return hf.Sum(nil), nil
}

// hashScaleSets returns the set of desired state scale set hashes
// Get certificate from the vault and tag on the end so it's included in
// the hash. BUT don't include the private key as:
//
// A quirk with PEM is that you don't get byte-for-byte what you sent necessarily
//
// In a given RSA key, there are multiple suitable values of 'D'.
// We send one, and sometimes MSFT returns another
// But the returned key validates, I can encrypt something with the sent
// key and decrypt it with the received key and visa-versa.
//
// Note: Azure Key Vault sometimes returns us a different value of D in the
// private key to the one we sent.  I don't believe this is a problem, but
// just don't expect reflect.DeepEqual(key.D, key2.D) to be true.
// References: https://stackoverflow.com/a/14233140,
// https://crypto.stackexchange.com/a/46572.
func (h *hasher) HashScaleSet(cs *api.OpenShiftManagedCluster, app *api.AgentPoolProfile) ([]byte, error) {
	// the hash is invariant of name, suffix, count
	appCopy := *app
	appCopy.Count = 0
	appCopy.Name = ""

	vmss, err := arm.Vmss(&h.pluginConfig, cs, &appCopy, "", "") // TODO: backupBlob is rather a layering violation here
	if err != nil {
		return nil, err
	}

	cert1, err := h.getCertificateFromVault(context.Background(), cs, cs.Properties.APICertProfile.KeyVaultSecretURL)
	if err != nil {
		return nil, err
	}
	cert2, err := h.getCertificateFromVault(context.Background(), cs, cs.Properties.RouterProfiles[0].RouterCertProfile.KeyVaultSecretURL)
	if err != nil {
		return nil, err
	}

	return hashVMSS(vmss, cert1, cert2)
}
