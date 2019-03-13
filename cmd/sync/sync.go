package main

import (
	"context"
	"flag"
	"time"

	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/addons"
	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/api/validate"
	"github.com/openshift/openshift-azure/pkg/cluster"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	azureclientstorage "github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
	"github.com/openshift/openshift-azure/pkg/util/configblob"
	"github.com/openshift/openshift-azure/pkg/util/log"
	"github.com/openshift/openshift-azure/pkg/util/vault"
)

var (
	dryRun    = flag.Bool("dry-run", false, "Print resources to be synced instead of mutating cluster state.")
	once      = flag.Bool("run-once", false, "If true, run only once then quit.")
	interval  = flag.Duration("interval", 3*time.Minute, "How often the sync process going to be rerun.")
	logLevel  = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")
	gitCommit = "unknown"
)

type sync struct {
	azs  azureclient.AccountsClient
	kvc  azureclient.KeyVaultClient
	blob azureclientstorage.Blob
	log  *logrus.Entry
}

func (s *sync) init(ctx context.Context, log *logrus.Entry) error {
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

	s.blob = bsc.GetContainerReference(cluster.ConfigContainerName).GetBlobReference(cluster.ConfigBlobName)

	s.log = log

	return nil
}

func (s *sync) getBlob(ctx context.Context) (*api.OpenShiftManagedCluster, error) {
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

	err = addons.EnrichCSStorageAccountKeys(ctx, s.azs, cs)
	if err != nil {
		return nil, err
	}

	v := validate.NewAPIValidator(cs.Config.RunningUnderTest)
	if errs := v.Validate(cs, nil, false); len(errs) > 0 {
		return nil, kerrors.NewAggregate(errs)
	}

	return cs, nil
}

func run(ctx context.Context, log *logrus.Entry) error {
	var s sync

	err := s.init(ctx, log)
	if err != nil {
		return err
	}

	t := time.NewTicker(*interval)

	var cs *api.OpenShiftManagedCluster
	for {
		cs, err = s.getBlob(ctx)
		if err == nil {
			break
		}
		log.Printf("s.getBlob: %s", err)
		<-t.C
	}

	for {
		log.Print("starting sync")
		if err := addons.Main(ctx, s.log, cs, *dryRun); err != nil {
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
