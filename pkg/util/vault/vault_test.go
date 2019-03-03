package vault

import (
	"testing"
)

func TestSplitSecretURL(t *testing.T) {
	secretURL := "https://myvault.vault.azure.net/secrets/mysecret"
	wantVaultURL := "https://myvault.vault.azure.net"
	wantSecretName := "mysecret"

	vaultURL, secretName, err := splitSecretURL(secretURL)
	if err != nil {
		t.Fatal(err)
	}

	if vaultURL != wantVaultURL {
		t.Errorf("got vaultURL %q, wanted %q", vaultURL, wantVaultURL)
	}

	if secretName != wantSecretName {
		t.Errorf("got secretName %q, wanted %q", secretName, wantSecretName)
	}
}
