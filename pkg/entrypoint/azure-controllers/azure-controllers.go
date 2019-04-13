package azurecontrollers

import (
	"context"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	"github.com/openshift/openshift-azure/pkg/controllers/customeradmin"
	"github.com/openshift/openshift-azure/pkg/util/log"
)

func start(cfg *Config) error {
	ctx := context.Background()
	logrus.SetLevel(log.SanitizeLogLevel(cfg.LogLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log := logrus.NewEntry(logrus.StandardLogger())

	log.Print("azure-controller pod starting")

	// TODO: Expose metrics port after SDK uses controller-runtime's dynamic client
	// sdk.ExposeMetricsPort()

	kCfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	m, err := manager.New(kCfg, manager.Options{})
	if err != nil {
		return err
	}

	stopCh := signals.SetupSignalHandler()

	if err := customeradmin.AddToManager(ctx, log, m, stopCh); err != nil {
		return err
	}

	log.Print("starting manager")
	return m.Start(stopCh)
}
