package main

import (
	"context"
	"flag"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/plugin"
	"github.com/openshift/openshift-azure/pkg/validate"
)

var logLevel = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")

// checks and sanitizes logLevel input
func sanitizeLogLevel(lvl *string) logrus.Level {
	switch strings.ToLower(*lvl) {
	case "debug":
		return logrus.DebugLevel
	case "info":
		return logrus.InfoLevel
	case "warning":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	default:
		// silently default to info
		return logrus.InfoLevel
	}
}

// healthCheck should get rolled into the end of createorupdate once the sync
// pod runs in the cluster
func healthCheck() error {
	// mock logger configuration
	logger := logrus.New()
	// sanitize input to only accept specific log levels and tolerate junk
	logger.SetLevel(sanitizeLogLevel(logLevel))
	entry := logrus.NewEntry(logger)
	entry = entry.WithFields(logrus.Fields{"resourceGroup": os.Getenv("RESOURCEGROUP")})

	// instantiate the plugin
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

	if os.Getenv("RUN_SYNC_LOCAL") != "true" {
		err = p.EnsureSyncPod(context.Background(), cs)
		if err != nil {
			return errors.Wrap(err, "cannot ensure sync pod")
		}
	}

	return p.HealthCheck(context.Background(), cs)
}

func main() {
	if err := healthCheck(); err != nil {
		panic(err)
	}
}
