package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"sort"
	"sync/atomic"
	"time"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/api/validate"
	"github.com/openshift/openshift-azure/pkg/cluster"
	"github.com/openshift/openshift-azure/pkg/sync"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	azureclientstorage "github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
	"github.com/openshift/openshift-azure/pkg/util/configblob"
	"github.com/openshift/openshift-azure/pkg/util/log"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
	"github.com/openshift/openshift-azure/pkg/util/vault"
)

var (
	dryRun    = flag.Bool("dry-run", false, "Print resources to be synced instead of mutating cluster state.")
	once      = flag.Bool("run-once", false, "If true, run only once then quit.")
	interval  = flag.Duration("interval", 3*time.Minute, "How often the sync process going to be rerun.")
	logLevel  = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")
	gitCommit = "unknown"
)

type syncer struct {
	azs   azureclient.AccountsClient
	kvc   azureclient.KeyVaultClient
	blob  azureclientstorage.Blob
	kc    kubernetes.Interface
	log   *logrus.Entry
	cs    *api.OpenShiftManagedCluster
	db    map[string]unstructured.Unstructured
	ready atomic.Value
}

func (s *syncer) init(ctx context.Context, log *logrus.Entry) error {
	s.ready.Store(false)

	cpc, err := cloudprovider.Load("_data/_out/azure.conf")
	if err != nil {
		return err
	}

	authorizer, err := azureclient.NewAuthorizer(cpc.AadClientID, cpc.AadClientSecret, cpc.TenantID, "")
	if err != nil {
		return err
	}

	s.azs = azureclient.NewAccountsClient(ctx, cpc.SubscriptionID, authorizer)

	vaultauthorizer, err := azureclient.NewAuthorizer(cpc.AadClientID, cpc.AadClientSecret, cpc.TenantID, azureclient.KeyVaultEndpoint)
	if err != nil {
		return err
	}

	s.kvc = azureclient.NewKeyVaultClient(ctx, vaultauthorizer)

	bsc, err := configblob.GetService(ctx, cpc)
	if err != nil {
		return err
	}

	s.blob = bsc.GetContainerReference(cluster.ConfigContainerName).GetBlobReference(cluster.SyncBlobName)

	s.log = log

	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		return err
	}

	mux := &http.ServeMux{}
	mux.Handle("/healthz/ready", http.HandlerFunc(s.readyHandler))

	go http.Serve(l, mux)

	return nil
}

func (s *syncer) readyHandler(w http.ResponseWriter, r *http.Request) {
	var errs []error

	if !s.ready.Load().(bool) {
		errs = []error{fmt.Errorf("sync pod has not completed first run")}
	} else {
		errs = sync.CalculateReadiness(s.kc, s.db, s.cs)
	}

	if len(errs) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-type", "text/plain")
	w.WriteHeader(http.StatusInternalServerError)
	for _, err := range errs {
		fmt.Fprintln(w, err)
	}
}

func (s *syncer) getBlob(ctx context.Context) (*api.OpenShiftManagedCluster, error) {
	s.log.Print("fetching config blob")
	cs, err := configblob.GetBlob(s.blob)
	if err != nil {
		return nil, err
	}

	s.log.Print("enriching config blob")
	err = vault.EnrichCSFromVault(ctx, s.kvc, cs)
	if err != nil {
		return nil, err
	}

	err = sync.EnrichCSStorageAccountKeys(ctx, s.azs, cs)
	if err != nil {
		return nil, err
	}

	v := validate.NewAPIValidator(cs.Config.RunningUnderTest)
	if errs := v.Validate(cs, nil, false); len(errs) > 0 {
		return nil, kerrors.NewAggregate(errs)
	}

	restconfig, err := managedcluster.RestConfigFromV1Config(cs.Config.AdminKubeconfig)
	if err != nil {
		return nil, err
	}

	s.kc, err = kubernetes.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

func (s *syncer) printDB() error {
	// impose an order to improve debuggability.
	var keys []string
	for k := range s.db {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		b, err := yaml.Marshal(s.db[k].Object)
		if err != nil {
			return err
		}

		s.log.Info(string(b))
	}

	return nil
}

func run(ctx context.Context, log *logrus.Entry) error {
	var s syncer

	err := s.init(ctx, log)
	if err != nil {
		return err
	}

	t := time.NewTicker(*interval)

	for {
		s.cs, err = s.getBlob(ctx)
		if err == nil {
			break
		}
		log.Printf("s.getBlob: %s", err)
		<-t.C
	}

	s.db, err = sync.ReadDB(s.cs)
	if err != nil {
		return err
	}

	if *dryRun {
		return s.printDB()
	}

	for {
		log.Print("starting sync")
		if err := sync.Main(ctx, s.log, s.cs, s.db, &s.ready); err != nil {
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
