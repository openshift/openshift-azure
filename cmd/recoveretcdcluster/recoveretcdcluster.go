package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster"
	"github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/fakerp/shared"
	"github.com/openshift/openshift-azure/pkg/plugin"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
	"github.com/openshift/openshift-azure/pkg/util/configblob"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

// this file will go away in the future, in favour of an admin action

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log := logrus.NewEntry(logrus.StandardLogger())

	if len(os.Args) != 2 {
		log.Infof("usage: %s blobname", os.Args[0])
	}

	dataDir, err := shared.FindDirectory(shared.DataDirectory)
	if err != nil {
		log.Fatal(err)
	}
	cs, err := managedcluster.ReadConfig(filepath.Join(dataDir, "containerservice.yaml"))
	if err != nil {
		log.Fatal(err)
	}

	cpc := &cloudprovider.Config{
		TenantID:        cs.Properties.AzProfile.TenantID,
		SubscriptionID:  cs.Properties.AzProfile.SubscriptionID,
		AadClientID:     cs.Properties.ServicePrincipalProfile.ClientID,
		AadClientSecret: cs.Properties.ServicePrincipalProfile.Secret,
		ResourceGroup:   cs.Properties.AzProfile.ResourceGroup,
	}

	ctx := context.Background()

	bsc, err := configblob.GetService(ctx, cpc)
	if err != nil {
		log.Fatal(err)
	}
	etcdContainer := bsc.GetContainerReference(cluster.EtcdBackupContainerName)

	var blobName string
	var exists bool
	if len(os.Args) == 2 {
		blobName = os.Args[1]

		blob := etcdContainer.GetBlobReference(blobName)
		exists, err = blob.Exists()
		if err != nil {
			log.Fatal(err)
		}
		if !exists {
			log.Infof("blob %q does not exist", blobName)
		}
	}

	if !exists {
		resp, err := etcdContainer.ListBlobs(storage.ListBlobsParameters{})
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("available blobs:")
		for _, blob := range resp.Blobs {
			log.Infof("  %s", blob.Name)
		}
		log.Fatal("exiting")
	}

	config, err := fakerp.GetPluginConfig()
	if err != nil {
		log.Fatal(err)
	}
	p, errs := plugin.NewPlugin(log, config)
	if len(errs) > 0 {
		log.Fatal(errs)
	}

	ctx = context.WithValue(ctx, api.ContextKeyClientID, cs.Properties.ServicePrincipalProfile.ClientID)
	ctx = context.WithValue(ctx, api.ContextKeyClientSecret, cs.Properties.ServicePrincipalProfile.Secret)
	ctx = context.WithValue(ctx, api.ContextKeyTenantID, cs.Properties.AzProfile.TenantID)

	deployer := fakerp.GetDeployer(log, cs, config)
	if err := p.RecoverEtcdCluster(ctx, cs, deployer, blobName); err != nil {
		log.Fatal(err)
	}

	log.Info("recovered cluster")
}
