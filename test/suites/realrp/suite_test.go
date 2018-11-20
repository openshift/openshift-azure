//+build e2e

package realrp

import (
	"flag"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/openshift-azure/test/util/client/azure"
)

var _ = BeforeSuite(func() {
	az = azure.NewClient()
})

func TestE2eRealRP(t *testing.T) {
	flag.Parse()
	fmt.Printf("Azure resource provider E2E tests starting, git commit %s\n", gitCommit)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Azure resource provider E2E Suite")
}
