//+build e2e

package azurereader

import (
	"flag"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/openshift-azure/test/util/client/kubernetes"
)

var _ = BeforeSuite(func() {
	kc = kubernetes.NewClient(*artifactDir)
})

func TestE2eRP(t *testing.T) {
	flag.Parse()
	fmt.Printf("Azure Cluster Reader E2E tests starting, git commit %s\n", gitCommit)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Azure Cluster Reader E2E Suite")
}
