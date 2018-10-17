package addons

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestNeedsUpdate(t *testing.T) {
	tests := []struct {
		name     string
		existing *unstructured.Unstructured
		updated  *unstructured.Unstructured
		exp      bool
	}{
		{
			name:     "no need for upgrade",
			existing: getObjectFromFile("testdata/service1.yaml"),
			updated:  getObjectFromFile("testdata/service2.yaml"),
			exp:      false,
		},
		{
			name:     "needs upgrade",
			existing: getObjectFromFile("testdata/service1.yaml"),
			updated:  getObjectFromFile("testdata/service3.yaml"),
			exp:      true,
		},
		{
			name:     "secret diff omitted",
			existing: getObjectFromFile("testdata/secret2.yaml"),
			updated:  getObjectFromFile("testdata/secret3.yaml"),
			exp:      true,
		},
	}

	for _, test := range tests {
		if got := needsUpdate(test.existing, test.updated); got != test.exp {
			t.Errorf("%s: expected update %t, got %t", test.name, test.exp, got)
		}
	}

}
