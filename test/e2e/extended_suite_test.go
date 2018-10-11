//+build e2e

package e2e

import (
	"flag"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	gitCommit  = "unknown"
	kubeconfig = flag.String("kubeconfig", "../../_data/_out/admin.kubeconfig", "Location of the kubeconfig")
)

var _ = BeforeSuite(func() {
	c = newTestClient(*kubeconfig)
})

func TestExtended(t *testing.T) {
	flag.Parse()
	fmt.Printf("e2e tests starting, git commit %s\n", gitCommit)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Extended Suite")
}
