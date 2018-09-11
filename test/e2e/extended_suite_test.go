//+build e2e

package e2e

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestExtended(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Extended Suite")
}
