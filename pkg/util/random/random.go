package random

import (
	"crypto/rand"
	"io"
	"math/big"
	"strings"
)

// String returns a random string of length n comprised of bytes in letterBytes
func String(letterBytes string, n int) (string, error) {
	b := make([]byte, n)
	for i := range b {
		o, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterBytes))))
		if err != nil {
			return "", err
		}
		b[i] = letterBytes[o.Int64()]
	}

	return string(b), nil
}

// AlphanumericString returns a random string of length n comprised of
// [A-Za-z0-9]
func AlphanumericString(n int) (string, error) {
	return String("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", n)
}

// LowerCaseAlphanumericString returns a random string of length n comprised of
// [a-z0-9]
func LowerCaseAlphanumericString(n int) (string, error) {
	return String("abcdefghijklmnopqrstuvwxyz0123456789", n)
}

// LowerCaseAlphaString returns a random string of length n comprised of [a-z]
func LowerCaseAlphaString(n int) (string, error) {
	return String("abcdefghijklmnopqrstuvwxyz", n)
}

// Bytes returns a random byte slice of legth n
func Bytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, err
	}
	return b, nil
}

func isMicrosoftReserved(s string) bool {
	// https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-manager-reserved-resource-name
	for _, reserved := range []string{"login", "microsoft", "windows", "xbox"} {
		if strings.Contains(strings.ToLower(s), reserved) {
			return true
		}
	}
	return false
}

// StorageAccountName returns a random string suitable for use as an Azure
// storage account name
func StorageAccountName(prefix string) (string, error) {
	for {
		name, err := LowerCaseAlphanumericString(24 - len(prefix))
		if err != nil {
			return "", err
		}
		if !isMicrosoftReserved(name) {
			return prefix + name, nil
		}
	}
}

// VaultURL returns a random string suitable for use as an Azure key vault URL
func VaultURL(prefix string) (string, error) {
	for {
		fqdn, err := FQDN("vault.azure.net", 24-len(prefix))
		if err != nil {
			return "", err
		}
		if !isMicrosoftReserved(fqdn) {
			return "https://" + prefix + fqdn, nil
		}
	}
}

// FQDN returns a random fully qualified domain name within a given parent
// domain
func FQDN(parentDomain string, n int) (string, error) {
	for {
		d, err := LowerCaseAlphaString(n)
		if err != nil {
			return "", err
		}
		if !isMicrosoftReserved(d) {
			return d + "." + parentDomain, nil
		}
	}
}
