//+build e2erp

package e2erp

import (
	"flag"
	"fmt"
	"testing"

	"github.com/kelseyhightower/envconfig"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	c         *testClient
	azureConf AzureConfig
	gitCommit = "unknown"
	manifest  = flag.String("manifest", "../../_data/manifest.yaml", "Path to the manifest to send to the RP")
)

var _ = BeforeSuite(func() {
	err := envconfig.Process("", &azureConf)
	if err != nil {
		panic(err)
	}
	c = newTestClient(azureConf)
})

func TestE2eRP(t *testing.T) {
	flag.Parse()
	fmt.Printf("E2E Resource Provider tests starting, git commit %s\n", gitCommit)
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Resource Provider Suite")
}
