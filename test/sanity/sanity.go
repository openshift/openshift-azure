package sanity

import (
	"context"
	"io"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/log"
	"github.com/openshift/openshift-azure/test/e2e/standard"
)

// Checker is a singleton sanity checker which is used by all the Ginkgo tests
var Checker *standard.SanityChecker
var GlobalLogger io.Writer

func init() {
	logrus.SetLevel(log.SanitizeLogLevel("Debug"))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logrus.SetOutput(GlobalLogger)
	log := logrus.NewEntry(logrus.StandardLogger())

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

	Checker, err = standard.NewSanityChecker(context.Background(), log, cs)
	if err != nil {
		panic(err)
	}
}
