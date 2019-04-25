package v20180930preview

import (
	"reflect"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-test/deep"
)

var sampleManagedCluster = &OpenShiftManagedCluster{
	Properties: &Properties{
		RouterProfiles: []RouterProfile{
			{
				Name:            to.StringPtr("Properties.RouterProfiles[0].Name"),
				PublicSubdomain: to.StringPtr("NewPublicSubdomain"),
			},
		},
	},
}

func TestDefaults(t *testing.T) {
	tests := []struct {
		name     string
		input    *OpenShiftManagedCluster
		expected *OpenShiftManagedCluster
	}{
		{
			name:  "sets default RouterProfile",
			input: &OpenShiftManagedCluster{},
			expected: &OpenShiftManagedCluster{
				Properties: &Properties{
					RouterProfiles: []RouterProfile{
						{
							Name: to.StringPtr("default"),
						},
					},
				},
			},
		},
		{
			name:     "sets no defaults",
			input:    sampleManagedCluster,
			expected: sampleManagedCluster,
		},
	}

	for _, test := range tests {
		setDefaults(test.input)

		if !reflect.DeepEqual(test.input, test.expected) {
			t.Errorf("%s: unexpected diff %s", test.name, deep.Equal(test.input, test.expected))
		}
	}
}
