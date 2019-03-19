package shared

import (
	"os"
)

const (
	LocalHttpAddr = "localhost:8080"
)

// IsUpdate return whether or not this is an update or create.
func IsUpdate() bool {
	_, err := os.Stat("_data/containerservice.yaml")
	return err == nil
}
