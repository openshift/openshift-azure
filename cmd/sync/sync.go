package main

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/addons"
	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/api/validate"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
	"github.com/openshift/openshift-azure/pkg/util/log"
)

var (
	dryRun    = flag.Bool("dry-run", false, "Print resources to be synced instead of mutating cluster state.")
	once      = flag.Bool("run-once", false, "If true, run only once then quit.")
	interval  = flag.Duration("interval", 3*time.Minute, "How often the sync process going to be rerun.")
	logLevel  = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")
	gitCommit = "unknown"
)

type sync struct {
	azs azureclient.AccountsClient
	log *logrus.Entry
}

func (s *sync) init(ctx context.Context, log *logrus.Entry) error {
	cpc, err := cloudprovider.Load("_data/_out/azure.conf")
	if err != nil {
		return err
	}

	authorizer, err := azureclient.NewAuthorizer(cpc.AadClientID, cpc.AadClientSecret, cpc.TenantID)
	if err != nil {
		return err
	}

	s.azs = azureclient.NewAccountsClient(ctx, cpc.SubscriptionID, authorizer)

	s.log = log

	return nil
}

// sync syncs the current state of the cluster with the
// desired state that is kept in a blob in an Azure storage
// account. It returns whether it managed to access the
// config blob or not and any error that occured.
func (s *sync) sync(ctx context.Context, log *logrus.Entry) error {
	s.log.Print("Sync process started")

	s.log.Print("reading config")
	b, err := ioutil.ReadFile("_data/_out/containerservice.json")
	if err != nil {
		return err
	}

	var cs *api.OpenShiftManagedCluster
	if err := json.Unmarshal(b, &cs); err != nil {
		return err
	}

	v := validate.NewAPIValidator(cs.Config.RunningUnderTest)
	if errs := v.Validate(cs, nil, false); len(errs) > 0 {
		return errors.Wrap(kerrors.NewAggregate(errs), "cannot validate _data/manifest.yaml")
	}

	if err := addons.Main(ctx, s.log, cs, s.azs, *dryRun); err != nil {
		return errors.Wrap(err, "cannot sync cluster config")
	}

	s.log.Print("Sync process complete")
	return nil
}

func main() {
	flag.Parse()
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	logger.SetLevel(log.SanitizeLogLevel(*logLevel))
	log := logrus.NewEntry(logger)
	log.Printf("sync pod starting, git commit %s", gitCommit)

	s := new(sync)
	ctx := context.Background()

	if err := s.init(ctx, log); err != nil {
		log.Fatalf("Cannot initialize sync: %v", err)
	}

	for {
		err := s.sync(ctx, log)
		if err != nil {
			log.Printf("Error while syncing: %v", err)
		}
		if *once {
			return
		}
		<-time.After(*interval)
	}
}
