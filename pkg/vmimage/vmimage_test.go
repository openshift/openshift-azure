package vmimage

import (
	"crypto/rand"
	"crypto/rsa"
	"os"
	"testing"
	"time"

	"github.com/openshift/openshift-azure/test/util/tls"
)

func TestGenerateTemplate(t *testing.T) {
	timestamp := time.Now().UTC().Format("200601021504")
	image := "rhel7-3.11-" + timestamp
	sshKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Error(err)
	}

	builder := Builder{
		SubscriptionID:      os.Getenv("AZURE_SUBSCRIPTION_ID"),
		Location:            os.Getenv("AZURE_REGION"),
		BuildResourceGroup:  "imagebuilder",
		Image:               image,
		ImageResourceGroup:  "images",
		ImageStorageAccount: "openshiftimages",
		ImageContainer:      "images",
		SSHKey:              sshKey,
		ClientKey:           tls.GetDummyPrivateKey(),
		ClientCert:          tls.GetDummyCertificate(),
	}
	_, err = builder.generateTemplate()
	if err != nil {
		t.Error(err)
	}
}
