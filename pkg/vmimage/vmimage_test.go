package vmimage

import (
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/util/cmp"
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

func TestValidateFields(t *testing.T) {
	tests := []struct {
		name             string
		builder          *Builder
		expectedContains string
	}{
		{
			name: "has custom image reference",
			builder: &Builder{
				Image:               "rhel7-3.11-197001010000",
				ImageResourceGroup:  "images",
				ImageStorageAccount: "foo",
				ImageContainer:      "images",
			},
		},
		{
			name: "has marketplace image reference",
			builder: &Builder{
				ImageSku:     "osa_111",
				ImageVersion: "latest",
			},
		},
		{
			name: "has marketplace and custom image reference",
			builder: &Builder{
				Image:               "rhel7-3.11-197001010000",
				ImageResourceGroup:  "images",
				ImageStorageAccount: "foo",
				ImageContainer:      "images",
				ImageSku:            "osa_111",
				ImageVersion:        "latest",
			},
			expectedContains: "confilicting fields",
		},
		{
			name: "has incomplete marketplace image reference",
			builder: &Builder{
				ImageSku: "osa_111",
			},
			expectedContains: "missing fields",
		},
		{
			name: "has incomplete custom image reference",
			builder: &Builder{
				ImageResourceGroup:  "images",
				ImageStorageAccount: "foo",
				ImageContainer:      "images",
			},
			expectedContains: "missing fields",
		},
	}

	for _, test := range tests {
		err := test.builder.ValidateFields()

		if err != nil && test.expectedContains == "" {
			t.Errorf("%s: unexpected error %#v", test.name, err.Error())
		}

		if err == nil && test.expectedContains != "" {
			t.Errorf("%s: expected error to contain %#v, got none", test.name, test.expectedContains)
		}

		if err != nil && !strings.Contains(err.Error(), test.expectedContains) {
			t.Errorf("%s: expected error to contain %#v, got error: %#v", test.name, test.expectedContains, err.Error())
		}
	}
}

func TestVMImageReference(t *testing.T) {
	tests := []struct {
		name     string
		builder  *Builder
		expected compute.ImageReference
	}{
		{
			name: "vm image to create a vm that we are going to use to build a new vm image",
			builder: &Builder{
				Image:               "rhel7-3.11-197001010000",
				ImageResourceGroup:  "images",
				ImageStorageAccount: "foo",
				ImageContainer:      "images",
			},
			expected: compute.ImageReference{
				Publisher: to.StringPtr("RedHat"),
				Offer:     to.StringPtr("RHEL"),
				Sku:       to.StringPtr("7-RAW"),
				Version:   to.StringPtr("latest"),
			},
		},
		{
			name: "custom vm image to validation",
			builder: &Builder{
				Image:               "rhel7-3.11-197001010000",
				ImageResourceGroup:  "images",
				ImageStorageAccount: "foo",
				ImageContainer:      "images",
				Validate:            true,
			},
			expected: compute.ImageReference{
				ID: to.StringPtr("/subscriptions//resourceGroups/images/providers/Microsoft.Compute/images/rhel7-3.11-197001010000"),
			},
		},
		{
			name: "marketplace vm image to validation",
			builder: &Builder{
				ImageSku:     "osa_111",
				ImageVersion: "latest",
				Validate:     true,
			},
			expected: compute.ImageReference{
				Publisher: to.StringPtr("redhat"),
				Offer:     to.StringPtr("osa"),
				Sku:       to.StringPtr("osa_111"),
				Version:   to.StringPtr("latest"),
			},
		},
	}

	for _, test := range tests {
		vmRef := test.builder.vmImageReference()

		diff := cmp.Diff(*vmRef, test.expected)
		if diff != "" {
			t.Errorf("%s: unexpected vm image reference: %s", test.name, diff)
		}
	}
}
