//+build e2e

package e2e

import (
	"flag"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	gitCommit  = "unknown"
	kubeconfig = flag.String("kubeconfig", "../../_data/_out/admin.kubeconfig", "Location of the kubeconfig")
)

var _ = BeforeSuite(func() {
	c = newTestClient(*kubeconfig)
	namespace := nameGen.generate("e2e-test-")
	c.createProject(namespace)
})

var _ = AfterSuite(func() {
	c.cleanupProject(10 * time.Minute)
})

func TestExtended(t *testing.T) {
	flag.Parse()
	fmt.Printf("e2e tests starting, git commit %s\n", gitCommit)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Extended Suite")
}
