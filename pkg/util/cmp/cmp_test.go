package cmp

import (
	"crypto/rsa"
	"crypto/x509"
	"testing"

	"github.com/openshift/openshift-azure/pkg/util/tls"
	testtls "github.com/openshift/openshift-azure/test/util/tls"
)

func TestX509CertComparer(t *testing.T) {
	tests := []struct {
		name   string
		x, y   *x509.Certificate
		expect bool
	}{
		{
			name:   "both nil",
			x:      nil,
			y:      nil,
			expect: true,
		},
		{
			name:   "one nil: x",
			x:      nil,
			y:      &x509.Certificate{},
			expect: false,
		},
		{
			name:   "one nil: y",
			x:      &x509.Certificate{},
			y:      nil,
			expect: false,
		},
		{
			name:   "all non-nil and equal",
			x:      &x509.Certificate{Raw: []byte{1}},
			y:      &x509.Certificate{Raw: []byte{1}},
			expect: true,
		},
		{
			name:   "all non-nil and not equal",
			x:      &x509.Certificate{Raw: []byte{1}},
			y:      &x509.Certificate{Raw: []byte{2}},
			expect: false,
		},
	}

	for _, test := range tests {
		got := x509CertComparer(test.x, test.y)
		if got != test.expect {
			t.Errorf("%s: expected %#v got %#v", test.name, test.expect, got)
		}
	}
}

func TestRsaPrivateKeyComparer(t *testing.T) {
	pk, _ := tls.NewPrivateKey()
	tests := []struct {
		name   string
		x, y   *rsa.PrivateKey
		expect bool
	}{
		{
			name:   "both nil",
			x:      nil,
			y:      nil,
			expect: true,
		},
		{
			name:   "compare 2 rsa keys - match",
			x:      testtls.DummyPrivateKey,
			y:      testtls.DummyPrivateKey,
			expect: true,
		},
		{
			name:   "compare 2 rsa keys - missmatch",
			x:      testtls.DummyPrivateKey,
			y:      pk,
			expect: false,
		},
	}

	for _, test := range tests {
		got := rsaPrivateKeyComparer(test.x, test.y)
		if got != test.expect {
			t.Errorf("%s: expected %#v got %#v", test.name, test.expect, got)
		}
	}
}
