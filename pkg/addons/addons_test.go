package addons

import (
	"io/ioutil"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func getObjectFromFile(path string) *unstructured.Unstructured {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	o, err := Unmarshal(b)
	if err != nil {
		panic(err)
	}
	return &o
}

// TODO: We need fuzz testing
func TestGetHash(t *testing.T) {
	tests := []struct {
		name  string
		i1    *unstructured.Unstructured
		i2    *unstructured.Unstructured
		match bool
	}{
		{
			name:  "same object matches",
			i1:    getObjectFromFile("testdata/secret1.yaml"),
			i2:    getObjectFromFile("testdata/secret1.yaml"),
			match: true,
		},
		{
			name:  "different objects do not match",
			i1:    getObjectFromFile("testdata/secret1.yaml"),
			i2:    getObjectFromFile("testdata/secret2.yaml"),
			match: false,
		},
		{
			name:  "semantically same objects match",
			i1:    getObjectFromFile("testdata/secret1.yaml"),
			i2:    getObjectFromFile("testdata/secret3.yaml"),
			match: true,
		},
	}

	for _, test := range tests {
		first := getHash(test.i1)
		sec := getHash(test.i2)
		if test.match && first != sec {
			t.Errorf("%s: expected hashes to match", test.name)
		}
		if !test.match && first == sec {
			t.Errorf("%s: unexpected hashes match", test.name)
		}
	}
}
