package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
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

// TODO(charlesakalugwu): Add unit tests for the handling of these metrics once
//  the upstream library supports it
var (
	syncInfoGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sync_info",
			Help: "General information about the sync process.",
		},
		[]string{"plugin_version", "image", "period_seconds"},
	)

	syncErrorsCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "sync_errors_total",
			Help: "Total number of errors encountered during sync executions.",
		},
	)

	syncInFlightGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sync_executions_inflight",
			Help: "Number of sync executions in progress.",
		},
	)

	syncLastExecutedGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sync_last_executed",
			Help: "The last time a sync was executed.",
		},
	)

	syncDurationSummary = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Name: "sync_duration_seconds",
			Help: "The duration of sync executions.",
		},
	)
)

func init() {
	prometheus.MustRegister(syncInfoGauge)
	prometheus.MustRegister(syncErrorsCounter)
	prometheus.MustRegister(syncDurationSummary)
	prometheus.MustRegister(syncInFlightGauge)
	prometheus.MustRegister(syncLastExecutedGauge)
}

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
	syncInfoGauge.With(prometheus.Labels{
		"plugin_version": cs.Config.PluginVersion,
		"image":          cs.Config.Images.Sync,
		"period_seconds": fmt.Sprintf("%d", int(cfg.interval.Seconds())),
	}).Set(1)

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

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.httpPort))
	if err != nil {
		return err
	}

	mux := &http.ServeMux{}
	mux.Handle("/healthz/ready", http.HandlerFunc(s.ReadyHandler))
	mux.Handle(cfg.metricsEndpoint, promhttp.Handler())

	go http.Serve(l, mux)

	t := time.NewTicker(cfg.interval)
	for {
		log.Print("starting sync")
		startTime := time.Now()
		syncInFlightGauge.Inc()
		if err := s.Sync(ctx); err != nil {
			log.Printf("sync error: %s", err)
			syncErrorsCounter.Inc()
		} else {
			log.Print("sync done")
		}
		syncDurationSummary.Observe(time.Now().Sub(startTime).Seconds())
		syncLastExecutedGauge.SetToCurrentTime()
		syncInFlightGauge.Dec()
		if cfg.once {
			return nil
		}
		<-t.C
	}
}
