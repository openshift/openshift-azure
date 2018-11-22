//+build e2e

package keyrotation

import (
	"flag"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/openshift-azure/test/util/client/azure"
	"github.com/openshift/openshift-azure/test/util/client/kubernetes"
)

var _ = BeforeSuite(func() {
	kc = kubernetes.NewClient(*kubeconfig, *artifactDir)
	az = azure.NewClient()
})

func TestE2eRP(t *testing.T) {
	flag.Parse()
	fmt.Printf("Key Rotation E2E tests starting, git commit %s\n", gitCommit)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Key Rotation E2E Suite")
}
