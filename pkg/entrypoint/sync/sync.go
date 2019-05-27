package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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

var (
	infoGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sync_info",
			Help: "General information about the sync process.",
		},
		[]string{"plugin_version", "image", "period_seconds"},
	)

	errorsCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "sync_errors_total",
			Help: "Total number of errors.",
		},
	)

	inFlightGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "sync_executions_inflight",
			Help: "Number of sync executions in progress.",
		},
	)

	lastExecutedGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "sync_last_executed",
			Help: "The last time a sync was run.",
		},
	)

	durationSummary = promauto.NewSummary(
		prometheus.SummaryOpts{
			Name: "sync_duration_seconds",
			Help: "The duration of sync runs.",
		},
	)
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
	infoGauge.With(prometheus.Labels{"plugin_version": cs.Config.PluginVersion, "image": cs.Config.Images.Sync, "period_seconds": fmt.Sprintf("%f", cfg.interval.Seconds())}).Set(1)

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
		startTime := time.Now()
		log.Print("starting sync")
		inFlightGauge.Inc()
		if err := s.Sync(ctx); err != nil {
			errorsCounter.Inc()
			log.Printf("sync error: %s", err)
		} else {
			log.Print("sync done")
		}
		durationSummary.Observe(time.Now().Sub(startTime).Seconds())
		lastExecutedGauge.SetToCurrentTime()
		inFlightGauge.Dec()
		if cfg.once {
			return nil
		}
		<-t.C
	}
}
