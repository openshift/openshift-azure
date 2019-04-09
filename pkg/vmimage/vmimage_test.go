package vmimage

import (
	"testing"

	"github.com/openshift/openshift-azure/test/util/tls"
)

func TestGenerateTemplate(t *testing.T) {
	builder := Builder{
		SSHKey:     tls.DummyPrivateKey,
		ClientKey:  tls.DummyPrivateKey,
		ClientCert: tls.DummyCertificate,
	}
	_, err := builder.generateTemplate()
	if err != nil {
		t.Error(err)
	}
}
