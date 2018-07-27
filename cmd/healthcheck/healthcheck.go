package main

import (
	"context"
	"io/ioutil"

	acsapi "github.com/Azure/acs-engine/pkg/api"
	"github.com/Azure/acs-engine/pkg/api/osa/vlabs"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/plugin"
	"github.com/openshift/openshift-azure/pkg/validate"
)

// healthCheck should get rolled into the end of createorupdate once the sync
// pod runs in the cluster
func healthCheck() error {
	var p api.Plugin = &plugin.Plugin{}

	b, err := ioutil.ReadFile("_data/manifest.yaml")
	if err != nil {
		return err
	}
	var ext *vlabs.OpenShiftCluster
	err = yaml.Unmarshal(b, &ext)
	if err != nil {
		return errors.Wrap(err, "cannot unmarshal _data/manifest.yaml")
	}
	if errs := validate.OpenShiftCluster(ext); len(errs) > 0 {
		return errors.Wrap(kerrors.NewAggregate(errs), "cannot validate _data/manifest.yaml")
	}
	cs := acsapi.ConvertVLabsOpenShiftClusterToContainerService(ext)

	configBytes, err := ioutil.ReadFile("_data/config.yaml")
	if err != nil {
		return errors.Wrap(err, "cannot read _data/config.yaml")
	}

	return p.HealthCheck(context.Background(), cs, configBytes)
}

func main() {
	if err := healthCheck(); err != nil {
		panic(err)
	}
}
