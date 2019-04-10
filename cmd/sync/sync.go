package main

import (
	"context"
	"encoding/json"
	"flag"
	"net"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster"
	"github.com/openshift/openshift-azure/pkg/sync"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
	"github.com/openshift/openshift-azure/pkg/util/configblob"
	"github.com/openshift/openshift-azure/pkg/util/enrich"
	"github.com/openshift/openshift-azure/pkg/util/log"
)

var (
	dryRun    = flag.Bool("dry-run", false, "Print resources to be synced instead of mutating cluster state.")
	once      = flag.Bool("run-once", false, "If true, run only once then quit.")
	interval  = flag.Duration("interval", 3*time.Minute, "How often the sync process going to be rerun.")
	logLevel  = flag.String("loglevel", "Info", "valid values are Debug, Info, Warning, Error")
	gitCommit = "unknown"
)

func run(ctx context.Context, log *logrus.Entry) error {
	cpc, err := cloudprovider.Load("_data/_out/azure.conf")
	if err != nil {
		return err
	}

	authorizer, err := azureclient.NewAuthorizer(cpc.AadClientID, cpc.AadClientSecret, cpc.TenantID, "")
	if err != nil {
		return err
	}

	azs := azureclient.NewAccountsClient(ctx, cpc.SubscriptionID, authorizer)

	vaultauthorizer, err := azureclient.NewAuthorizer(cpc.AadClientID, cpc.AadClientSecret, cpc.TenantID, azureclient.KeyVaultEndpoint)
	if err != nil {
		return err
	}

	kvc := azureclient.NewKeyVaultClient(ctx, vaultauthorizer)

	bsc, err := configblob.GetService(ctx, cpc)
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

	log.Print("creating new sync")
	s, err := sync.New(log, cs, true)
	if err != nil {
		return err
	}

	if *dryRun {
		return s.PrintDB()
	}

	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		return err
	}

	mux := &http.ServeMux{}
	mux.Handle("/healthz/ready", http.HandlerFunc(s.ReadyHandler))

	go http.Serve(l, mux)

	t := time.NewTicker(*interval)
	for {
		log.Print("starting sync")
		if err := s.Sync(ctx); err != nil {
			log.Printf("sync error: %s", err)
		} else {
			log.Print("sync done")
		}
		if *once {
			return nil
		}
		<-t.C
	}
}

func main() {
	flag.Parse()
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	logger.SetLevel(log.SanitizeLogLevel(*logLevel))
	log := logrus.NewEntry(logger)
	log.Printf("sync pod starting, git commit %s", gitCommit)

	if err := run(context.Background(), log); err != nil {
		log.Fatal(err)
	}
}
