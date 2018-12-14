package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/fakerp/shared"
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

	ctx := context.Background()
	ctx = context.WithValue(ctx, api.ContextKeyClientID, os.Getenv("AZURE_CLIENT_ID"))
	ctx = context.WithValue(ctx, api.ContextKeyClientSecret, os.Getenv("AZURE_CLIENT_SECRET"))
	ctx = context.WithValue(ctx, api.ContextKeyTenantID, os.Getenv("AZURE_TENANT_ID"))

	config, err := fakerp.GetPluginConfig()
	if err != nil {
		log.Fatal(err)
	}
	p, errs := plugin.NewPlugin(log, config)
	if len(errs) > 0 {
		log.Fatal(kerrors.NewAggregate(errs))
	}
	dataDir, err := shared.FindDirectory(shared.DataDirectory)
	if err != nil {
		return
	}
	cs, err := managedcluster.ReadConfig(filepath.Join(dataDir, "containerservice.yaml"))
	if err != nil {
		log.Fatal(err)
	}

	deployer := fakerp.GetDeployer(log, cs, config)
	if err := p.RecoverEtcdCluster(ctx, cs, deployer, blobName); err != nil {
		log.Fatal(err)
	}

	log.Info("recovered cluster")
}
