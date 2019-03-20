package sync

import (
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/go-test/deep"
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
		i, err := Unmarshal(b)
		if err != nil {
			t.Error(err)
		}

		b, err = ioutil.ReadFile(strings.Replace(match, "-in.yaml", "-out.yaml", -1))
		if err != nil {
			t.Error(err)
		}
		o, err := Unmarshal(b)
		if err != nil {
			t.Error(err)
		}

		clean(i)
		if !reflect.DeepEqual(i, o) {
			t.Errorf("%s:\n%s", match, strings.Join(deep.Equal(i, o), "\n"))
		}
	}
}
