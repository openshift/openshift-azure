package main

import (
	"context"
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/plugin"
	"github.com/openshift/openshift-azure/pkg/validate"
)

// healthCheck should get rolled into the end of createorupdate once the sync
// pod runs in the cluster
func healthCheck() error {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	log := logrus.NewEntry(logger)

	// instantiate the plugin
	p := plugin.NewPlugin(log)

	b, err := ioutil.ReadFile("_data/containerservice.yaml")
	if err != nil {
		return err
	}
	var cs *acsapi.ContainerService
	err = yaml.Unmarshal(b, &cs)
	if err != nil {
		return errors.Wrap(err, "cannot unmarshal _data/containerservice.yaml")
	}

	if errs := validate.ContainerService(cs, nil); len(errs) > 0 {
		return errors.Wrap(kerrors.NewAggregate(errs), "cannot validate _data/containerservice.yaml")
	}

	return p.HealthCheck(context.Background(), cs)
}

func main() {
	if err := healthCheck(); err != nil {
		panic(err)
	}
}
