//+build e2e

package e2e

import (
	"flag"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	_ "github.com/openshift/openshift-azure/test/e2e/specs"
	_ "github.com/openshift/openshift-azure/test/e2e/specs/fakerp"
	_ "github.com/openshift/openshift-azure/test/e2e/specs/realrp"
)

var (
	gitCommit = "unknown"
)

func TestE2E(t *testing.T) {
	fmt.Printf("e2e tests starting, git commit %s\n", gitCommit)

	flag.Parse()

	RegisterFailHandler(Fail)
	RunSpecs(t, "e2e tests")
}
