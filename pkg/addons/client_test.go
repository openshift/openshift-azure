package addons

import (
	"testing"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var clientTests = []struct {
	name     string
	existing *unstructured.Unstructured
	updated  *unstructured.Unstructured
	exp      bool
	diff     bool
}{
	{
		name:     "no need for upgrade",
		existing: getObjectFromFile("testdata/service1.yaml"),
		updated:  getObjectFromFile("testdata/service2.yaml"),
		exp:      false, // no upgrade expected
		diff:     false, // no diff expected
	},
	{
		name:     "needs upgrade",
		existing: getObjectFromFile("testdata/service1.yaml"),
		updated:  getObjectFromFile("testdata/service3.yaml"),
		exp:      true, // upgrade expected
		diff:     true, // diff expected
	},
	{
		name:     "secret diff omitted",
		existing: getObjectFromFile("testdata/secret2.yaml"),
		updated:  getObjectFromFile("testdata/secret3.yaml"),
		exp:      true,  // upgrade expected
		diff:     false, // no diff expected
	},
}
var logger = logrus.New()
var entry = logrus.NewEntry(logger)

func TestNeedsUpdate(t *testing.T) {
	for _, test := range clientTests {
		if got := needsUpdate(test.existing, test.updated, entry); got != test.exp {
			t.Errorf("%s: expected update %t, got %t", test.name, test.exp, got)
		}
	}
}

func TestShouldPrintDiff(t *testing.T) {
	for _, test := range clientTests {
		if got := printDiff(test.existing, test.updated, entry); got != test.diff {
			t.Errorf("%s: expected to print diff %t, got %t", test.name, test.diff, got)
		}
	}
}
