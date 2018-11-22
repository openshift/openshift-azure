//+build e2e

package scaleupdown

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
	az = azure.NewClient()
	kc = kubernetes.NewClient(*kubeconfig, *artifactDir)
})

func TestE2eRealRP(t *testing.T) {
	flag.Parse()
	fmt.Printf("Scale Out/In E2E tests starting, git commit %s\n", gitCommit)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scale Out/In E2E Suite")
}
