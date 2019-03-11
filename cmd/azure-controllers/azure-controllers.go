package main

import (
	"context"
	"flag"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	"github.com/openshift/openshift-azure/pkg/controllers/customeradmin"
	"github.com/openshift/openshift-azure/pkg/util/log"
)

var (
	logLevel  = flag.String("loglevel", "Debug", "Valid values are Debug, Info, Warning, Error")
	gitCommit = "unknown"
)

func main() {
	ctx := context.Background()

	flag.Parse()
	logrus.SetLevel(log.SanitizeLogLevel(*logLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	log := logrus.NewEntry(logrus.StandardLogger())
	log.Printf("azure-controller pod starting, git commit %s", gitCommit)

	// TODO: Expose metrics port after SDK uses controller-runtime's dynamic client
	// sdk.ExposeMetricsPort()

	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	m, err := manager.New(cfg, manager.Options{})
	if err != nil {
		log.Fatal(err)
	}

	stopCh := signals.SetupSignalHandler()

	if err := customeradmin.AddToManager(ctx, log, m, stopCh); err != nil {
		log.Fatal(err)
	}

	log.Print("starting manager")
	log.Fatal(m.Start(stopCh))
}
