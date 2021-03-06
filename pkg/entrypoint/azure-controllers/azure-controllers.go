package azurecontrollers

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	"github.com/openshift/openshift-azure/pkg/controllers/customeradmin"
	"github.com/openshift/openshift-azure/pkg/util/log"
)

func start(cfg *cmdConfig) error {
	ctx := context.Background()
	logrus.SetLevel(log.SanitizeLogLevel(cfg.LogLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log := logrus.NewEntry(logrus.StandardLogger())

	log.Print("azure-controller pod starting")

	// TODO(charlesakalugwu): Use controller-runtime's metrics exposition when
	//  we are able to update controller-runtime to any versions > 0.1.0
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.httpPort))
	if err != nil {
		return err
	}

	mux := &http.ServeMux{}
	mux.Handle("/healthz/ready", http.HandlerFunc(readyHandler))
	mux.Handle(cfg.metricsEndpoint, customeradmin.MetricsHandler())

	go http.Serve(l, mux)

	managerConfig, err := config.GetConfig()
	if err != nil {
		return err
	}

	m, err := manager.New(managerConfig, manager.Options{})
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

func readyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
