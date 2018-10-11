//+build e2erp

package e2erp

import (
	"flag"
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	c         *testClient
	gitCommit = "unknown"
	manifest  = flag.String("manifest", "../../_data/manifest.yaml", "Path to the manifest to send to the RP")
)

var _ = BeforeSuite(func() {
	c = newTestClient(os.Getenv("RESOURCEGROUP"))
	if err := c.setup(*manifest); err != nil {
		panic(err)
	}
})

var _ = AfterSuite(func() {
	if err := c.teardown(); err != nil {
		panic(err)
	}
})

func TestE2eRP(t *testing.T) {
	flag.Parse()
	fmt.Printf("E2E Resource Provider tests starting, git commit %s\n", gitCommit)
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Resource Provider Suite")
}
