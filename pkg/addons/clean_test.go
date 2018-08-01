package addons

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/api/equality"
)

func TestClean(t *testing.T) {
	matches, err := filepath.Glob("testdata/clean/*-in.yaml")
	if err != nil {
		t.Fatal(err)
	}

	for _, match := range matches {
		b, err := ioutil.ReadFile(match)
		if err != nil {
			t.Error(err)
		}
		i, err := unmarshal(b)
		if err != nil {
			t.Error(err)
		}

		b, err = ioutil.ReadFile(strings.Replace(match, "-in.yaml", "-out.yaml", -1))
		if err != nil {
			t.Error(err)
		}
		o, err := unmarshal(b)
		if err != nil {
			t.Error(err)
		}

		Clean(i)
		if !reflect.DeepEqual(i, o) {
			t.Errorf("%s:\n%s", match, strings.Join(equality.Semantic.DeepEqual(i, o), "\n"))
		}
	}
}
