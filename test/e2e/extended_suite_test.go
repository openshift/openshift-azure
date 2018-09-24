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
	gitCommit = "unknown"
	config    = flag.String("config", "../../_data/containerservice.yaml", "Location of the config")
)

func TestExtended(t *testing.T) {
	flag.Parse()
	fmt.Printf("e2e tests starting, git commit %s\n", gitCommit)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Extended Suite")
}
