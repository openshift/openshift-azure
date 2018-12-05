package fakerp

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func basepath() string {
	return fmt.Sprintf("%s/%s", os.TempDir(), "dirwalk")
}

func pathTo(name string) string {
	return filepath.Join(basepath(), name)
}

func chDir(t *testing.T, name string) string {
	if err := os.Chdir(pathTo(name)); err != nil {
		t.Errorf("could not switch to directory: %v", err)
	}
	return name
}

func tmpDir(t *testing.T, name string) string {
	err := os.MkdirAll(pathTo(name), os.ModePerm)
	if err != nil {
		t.Errorf("could not create %s: %v", name, err)
	}
	return name
}

func cleanUp(t *testing.T, workingDir string) {
	if err := os.Chdir(workingDir); err != nil {
		t.Errorf("could not change directory to %s: %v", workingDir, err)
	}
	if err := os.RemoveAll(basepath()); err != nil {
		t.Errorf("could not remove base directory %s: %v", basepath(), err)
	}
}

func TestFindDirectory(t *testing.T) {
	wd, _ := os.Getwd()
	defer cleanUp(t, wd)

	tmpDir(t, "/directory/with/two/folders")
	tmpDir(t, "/directory/with/secret")
	chDir(t, "/directory/with/two")

	secretDir, err := FindDirectory("secret")
	if err != nil {
		t.Errorf("error finding directory: %v", err)
	}

	target := pathTo("/directory/with/secret")
	if secretDir != target {
		t.Errorf("expected %s, got %s", secretDir, target)
	}
}
