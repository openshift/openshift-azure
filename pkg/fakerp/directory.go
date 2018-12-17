package fakerp

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindDirectory returns the absolute path to a directory containing name within
// the directory hierarchy of the caller. The search starts at the current working
// directory of the caller
func FindDirectory(name string) (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not get working directory: %v", err)
	}
	return walkParents(current, name)
}

// walkParents takes a path to a directory and a folder name and attempts to recursively
// walk up the ancestors of the path in an attempt to find an ancestor containing name
func walkParents(path, name string) (string, error) {
	target := filepath.Join(path, name)
	if info, err := os.Stat(target); err == nil {
		if info.IsDir() {
			return target, nil
		}
	}
	if path == "/" {
		return "", fmt.Errorf("could not find %s in the directory hierarchy", name)
	}
	parent := filepath.Dir(path)
	return walkParents(parent, name)
}
