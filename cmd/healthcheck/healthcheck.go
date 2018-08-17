package main

import (
	"context"
	"flag"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/plugin"
	"github.com/openshift/openshift-azure/pkg/validate"
)

var logLevel = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")

// healthCheck should get rolled into the end of createorupdate once the sync
// pod runs in the cluster
func healthCheck() error {
	// mock logger configuration
	logger := logrus.New()
	// sanitize input to only accept specific log levels and tolerate junk
	logger.SetLevel(log.SanitizeLogLevel(*logLevel))

	// instantiate the plugin
	entry := logrus.NewEntry(logger).WithFields(logrus.Fields{"resourceGroup": os.Getenv("RESOURCEGROUP")})
	p := plugin.NewPlugin(entry)

	b, err := ioutil.ReadFile("_data/containerservice.yaml")
	if err != nil {
		return err
	}
	var cs *acsapi.OpenShiftManagedCluster
	err = yaml.Unmarshal(b, &cs)
	if err != nil {
		return errors.Wrap(err, "cannot unmarshal _data/containerservice.yaml")
	}

	if errs := validate.Validate(cs, nil, false); len(errs) > 0 {
		return errors.Wrap(kerrors.NewAggregate(errs), "cannot validate _data/containerservice.yaml")
	}

	return p.HealthCheck(context.Background(), cs)
}

func main() {
	if err := healthCheck(); err != nil {
		panic(err)
	}
}
