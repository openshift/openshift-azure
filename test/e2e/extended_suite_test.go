//+build e2e

package e2e

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var gitCommit = "unknown"

func TestExtended(t *testing.T) {
	fmt.Printf("e2e tests starting, git commit %s\n", gitCommit)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Extended Suite")
}
