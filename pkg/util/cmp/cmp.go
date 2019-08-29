package cmp

import (
	"crypto/x509"

	gocmp "github.com/google/go-cmp/cmp"
)

// Diff is a wrapper for github.com/google/go-cmp/cmp.Diff with extra options
func Diff(x, y interface{}, opts ...gocmp.Option) string {

	// FIXME: Remove x509CertComparer after upgrading to a Go version that includes https://github.com/golang/go/issues/28743
	opts = append(opts, gocmp.Comparer(x509CertComparer))

	return gocmp.Diff(x, y, opts...)
}

func x509CertComparer(x, y *x509.Certificate) bool {
	if x == nil || y == nil {
		return x == y
	}

	return x.Equal(y)
}
