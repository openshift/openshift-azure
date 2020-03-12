package sync

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranslatableData(t *testing.T) {
	var errors []string

	err := filepath.Walk("data", func(path string, info os.FileInfo, errIn error) error {
		if errIn != nil {
			return errIn
		}

		if info.IsDir() || !strings.HasSuffix(info.Name(), ".yaml") {
			return nil
		}

		// load the yaml file from the data directory
		b, err := ioutil.ReadFile(path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", path, err.Error()))
		}
		o, err := unmarshal(b)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", path, err.Error()))
		}

		parts := strings.Split(path, "/")
		kind := parts[1]
		// cluster-wide kinds have an empty string as namespace
		namespace := ""
		name := parts[2]
		// namespaced kinds have 4 directories depth, cluster-wide have 3
		if len(parts) == 4 {
			namespace = parts[2]
			name = parts[3]
		}
		name = strings.TrimSuffix(name, ".yaml")

		// check the namespace matches
		if namespace != o.GetNamespace() {
			errors = append(errors, fmt.Sprintf("%s namespace doesn't match its path (%s != %s)", path, namespace, o.GetNamespace()))
		}

		// check the kind is the prefix for the given template
		apiVersion := o.GetAPIVersion()
		oKind := o.GetKind()
		if strings.Contains(apiVersion, "/") {
			oKind = fmt.Sprintf("%s.%s", oKind, strings.Split(apiVersion, "/")[0])
		}
		if kind != oKind {
			errors = append(errors, fmt.Sprintf("%s kind doesn't match its path (%s != %s)", path, kind, oKind))
		}

		// check the name matches, caveat: all colons (":") are replaced with single dash ("-")
		oName := strings.Replace(o.GetName(), ":", "-", -1)
		if name != oName {
			errors = append(errors, fmt.Sprintf("%s name doesn't match its path (%s != %s)", path, name, oName))
		}

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	if len(errors) > 0 {
		t.Errorf("error in translatable data files: \n%s", strings.Join(errors, "\n"))
	}
}
