package sanity

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/test/e2e/standard"
	testlogger "github.com/openshift/openshift-azure/test/util/log"
)

// Checker is a singleton sanity checker which is used by all the Ginkgo tests
var Checker *standard.SanityChecker

func init() {
	if os.Getenv("TEST_IN_PRODUCTION") != "true" {
		b, err := ioutil.ReadFile("../../_data/containerservice.yaml") // running via `go test`
		if os.IsNotExist(err) {
			b, err = ioutil.ReadFile("_data/containerservice.yaml") // running via compiled test binary
		}
		if err != nil {
			panic(err)
		}

		var cs *api.OpenShiftManagedCluster
		if err := yaml.Unmarshal(b, &cs); err != nil {
			panic(err)
		}

		Checker, err = standard.NewSanityChecker(context.Background(), testlogger.GetTestLogger(), cs)
		if err != nil {
			panic(err)
		}
	}
}
