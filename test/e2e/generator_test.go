//+build e2e

package e2e

import (
	"fmt"

	utilrand "k8s.io/apimachinery/pkg/util/rand"
)

// simpleNameGenerator generates random names.
type simpleNameGenerator struct{}

// nameGen is a generator that returns the name plus a random suffix of five alphanumerics
// when a name is requested. The string is guaranteed to not exceed the length of a standard Kubernetes
// name (63 characters)
var nameGen = simpleNameGenerator{}

const (
	// Copied from k8s.io/apiserver/pkg/storage/names/generate.go
	maxNameLength          = 63
	randomLength           = 5
	maxGeneratedNameLength = maxNameLength - randomLength
)

func (simpleNameGenerator) generate(base string) string {
	if len(base) > maxGeneratedNameLength {
		base = base[:maxGeneratedNameLength]
	}
	return fmt.Sprintf("%s%s", base, utilrand.String(randomLength))
}
