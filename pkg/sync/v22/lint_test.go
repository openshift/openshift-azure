package sync

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
)

func TestLint(t *testing.T) {
	var paths []string

	err := filepath.Walk("data", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		b1, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		u, err := unmarshal(b1)
		if err != nil {
			return err
		}

		if err = clean(u); err != nil {
			return err
		}

		defaults(u)

		b2, err := yaml.Marshal(u.Object)
		if err != nil {
			return err
		}

		if _, found := os.LookupEnv("REGENERATE"); !found {
			if !bytes.Equal(b1, b2) {
				paths = append(paths, path)
			}
		} else {
			if err = ioutil.WriteFile(path, b2, 0666); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) > 0 {
		t.Errorf("invalid files (rerun with REGENERATE environment variable set):\n  %s", strings.Join(paths, "\n  "))
	}
}
