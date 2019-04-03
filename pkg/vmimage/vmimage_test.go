package vmimage

import (
	"testing"

	"github.com/openshift/openshift-azure/test/util/tls"
)

func TestGenerateTemplate(t *testing.T) {
	builder := Builder{
		SSHKey:     tls.GetDummyPrivateKey(),
		ClientKey:  tls.GetDummyPrivateKey(),
		ClientCert: tls.GetDummyCertificate(),
	}
	_, err := builder.generateTemplate()
	if err != nil {
		t.Error(err)
	}
}
