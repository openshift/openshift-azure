package shared

import (
	"os"
	"path/filepath"
)

const (
	AdminContext  = "/admin"
	DataDirectory = "_data/"
	LocalHttpAddr = "localhost:8080"
)

// IsUpdate return whether or not this is an update or create.
func IsUpdate() bool {
	dataDir, err := FindDirectory(DataDirectory)
	if err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(dataDir, "containerservice.yaml")); err == nil {
		return true
	}
	return false
}
