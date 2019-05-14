package sync

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster"
	syncapi "github.com/openshift/openshift-azure/pkg/sync"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/keyvault"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
	"github.com/openshift/openshift-azure/pkg/util/configblob"
	"github.com/openshift/openshift-azure/pkg/util/enrich"
	"github.com/openshift/openshift-azure/pkg/util/log"
)

func start(cfg *cmdConfig) error {
	ctx := context.Background()
	logrus.SetLevel(log.SanitizeLogLevel(cfg.LogLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log := logrus.NewEntry(logrus.StandardLogger())

	log.Print("sync pod starting")

	cpc, err := cloudprovider.Load("_data/_out/azure.conf")
	if err != nil {
		return err
	}

	authorizer, err := azureclient.NewAuthorizer(cpc.AadClientID, cpc.AadClientSecret, cpc.TenantID, "")
	if err != nil {
		return err
	}

	azs := storage.NewAccountsClient(ctx, log, cpc.SubscriptionID, authorizer)

	vaultauthorizer, err := azureclient.NewAuthorizer(cpc.AadClientID, cpc.AadClientSecret, cpc.TenantID, azureclient.KeyVaultEndpoint)
	if err != nil {
		return err
	}

	kvc := keyvault.NewKeyVaultClient(ctx, log, vaultauthorizer)

	bsc, err := configblob.GetService(ctx, log, cpc)
	if err != nil {
		return err
	}

	c := bsc.GetContainerReference(cluster.ConfigContainerName)
	blob := c.GetBlobReference(cluster.SyncBlobName)

	log.Print("reading config")
	rc, err := blob.Get(nil)
	if err != nil {
		return err
	}
	defer rc.Close()

	var cs *api.OpenShiftManagedCluster
	err = json.NewDecoder(rc).Decode(&cs)
	if err != nil {
		return err
	}
	log.Printf("running sync for plugin %s", cs.Config.PluginVersion)

	log.Print("enriching config")
	err = enrich.CertificatesFromVault(ctx, kvc, cs)
	if err != nil {
		return err
	}

	log.Print("enriching storage account keys")
	err = enrich.StorageAccountKeys(ctx, azs, cs)
	if err != nil {
		return err
	}

	s, err := syncapi.New(log, cs, true)
	if err != nil {
		return err
	}

	if cfg.dryRun {
		return s.PrintDB()
	}

	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		return err
	}

	mux := &http.ServeMux{}
	mux.Handle("/healthz/ready", http.HandlerFunc(s.ReadyHandler))
	mux.Handle("/metrics", promhttp.Handler())

	go http.Serve(l, mux)

	t := time.NewTicker(cfg.interval)
	for {
		log.Print("starting sync")
		if err := s.Sync(ctx); err != nil {
			log.Printf("sync error: %s", err)
		} else {
			log.Print("sync done")
		}
		if cfg.once {
			return nil
		}
		<-t.C
	}
}
