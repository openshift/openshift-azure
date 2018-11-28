package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/plugin"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log := logrus.NewEntry(logrus.StandardLogger())
	if len(os.Args) != 2 {
		log.Fatal("Usage recoveretcdcluster <blobname>")
	}
	blobName := os.Args[1]
	conf, err := fakerp.NewConfig(log, true)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, api.ContextKeyClientID, conf.ClientID)
	ctx = context.WithValue(ctx, api.ContextKeyClientSecret, conf.ClientSecret)
	ctx = context.WithValue(ctx, api.ContextKeyTenantID, conf.TenantID)

	config, err := fakerp.GetPluginConfig()
	if err != nil {
		log.Fatal(err)
	}
	p, errs := plugin.NewPlugin(log, config)
	if len(errs) > 0 {
		log.Fatal(kerrors.NewAggregate(errs))
	}
	dataDir, err := fakerp.FindDirectory(fakerp.DataDirectory)
	if err != nil {
		return
	}
	cs, err := managedcluster.ReadConfig(filepath.Join(dataDir, "containerservice.yaml"))
	if err != nil {
		log.Fatal(err)
	}

	deployer := fakerp.GetDeployer(cs, log, config)
	if err := p.RecoverEtcdCluster(ctx, cs, deployer, blobName); err != nil {
		log.Fatal(err)
	}

	log.Info("recovered cluster")
}
