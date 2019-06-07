package random

import (
	"crypto/rand"
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

func isMicrosoftReserved(s string) bool {
	// https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-manager-reserved-resource-name
	for _, reserved := range []string{"login", "microsoft", "windows", "xbox"} {
		if strings.Contains(strings.ToLower(s), reserved) {
			return true
		}
	}
	return false
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
