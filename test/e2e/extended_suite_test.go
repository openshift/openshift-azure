//+build e2e

package e2e

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
)

var (
	gitCommit   = "unknown"
	kubeconfig  = flag.String("kubeconfig", "../../_data/_out/admin.kubeconfig", "Location of the kubeconfig")
	artifactDir = flag.String("artifact-dir", "", "Directory to place artifacts when a test fails")
)

var _ = BeforeSuite(func() {
	c = newTestClient(*kubeconfig, *artifactDir)

	focus := []byte(config.GinkgoConfig.FocusString)
	if strings.Contains(string(focus), "\\[CustomerAdmin\\]") {
		_, err := os.Stat("../../_data/_out/customer-cluster-admin.kubeconfig")
		if err != nil {
			panic(err)
		}
		_, err = os.Stat("../../_data/_out/customer-cluster-reader.kubeconfig")
		if err != nil {
			panic(err)
		}
		creader = newTestClient("../../_data/_out/customer-cluster-reader.kubeconfig", *artifactDir)
		cadmin = newTestClient("../../_data/_out/customer-cluster-admin.kubeconfig", *artifactDir)
	}
})

func TestExtended(t *testing.T) {
	flag.Parse()
	fmt.Printf("e2e tests starting, git commit %s\n", gitCommit)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Extended Suite")
}
