//+build e2e

package customeradmin

import (
	"flag"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/openshift-azure/test/util/client/kubernetes"
)

var _ = BeforeSuite(func() {
	kc = kubernetes.NewClient(*kubeconfig, *artifactDir)
})

func TestE2eRP(t *testing.T) {
	flag.Parse()
	fmt.Printf("Customer Cluster Admin E2E tests starting, git commit %s\n", gitCommit)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Customer Cluster Admin E2E Suite")
}
