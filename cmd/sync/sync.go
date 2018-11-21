package main

import (
	"context"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/addons"
	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	azureclientstorage "github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
	"github.com/openshift/openshift-azure/pkg/util/configblob"
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
	azs  azureclient.AccountsClient
	blob azureclientstorage.Blob
}

func (s *sync) init(ctx context.Context) error {
	cpc, err := cloudprovider.Load("_data/_out/azure.conf")
	if err != nil {
		return err
	}

	authorizer, err := azureclient.NewAuthorizer(cpc.AadClientID, cpc.AadClientSecret, cpc.TenantID)
	if err != nil {
		return err
	}

	s.azs = azureclient.NewAccountsClient(cpc.SubscriptionID, authorizer, nil)

	bsc, err := configblob.GetService(ctx, cpc)
	if err != nil {
		return err
	}

	s.blob = bsc.GetContainerReference(cluster.ConfigContainerName).GetBlobReference(cluster.ConfigBlobName)

	return nil
}

func (s *sync) getBlob() (*api.OpenShiftManagedCluster, error) {
	logrus.Print("reading config blob")

	var rc io.ReadCloser
	var err error
	err = wait.PollImmediateInfinite(time.Second, func() (bool, error) {
		rc, err = s.blob.Get(nil)

		if err, ok := err.(azstorage.AzureStorageServiceError); ok && err.StatusCode == http.StatusNotFound {
			return false, nil
		}

		return err == nil, err
	})
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	logrus.Print("read config blob")

	b, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	var cs *api.OpenShiftManagedCluster
	if err := yaml.Unmarshal(b, &cs); err != nil {
		return nil, err
	}
	return cs, nil
}

// sync syncs the current state of the cluster with the
// desired state that is kept in a blob in an Azure storage
// account. It returns whether it managed to access the
// config blob or not and any error that occured.
func (s *sync) sync(ctx context.Context) (bool, error) {
	logrus.Print("Sync process started")
	cs, err := s.getBlob()
	if err != nil {
		return false, err
	}

	if errs := api.Validate(cs, nil, false); len(errs) > 0 {
		return true, errors.Wrap(kerrors.NewAggregate(errs), "cannot validate _data/manifest.yaml")
	}

	if err := addons.Main(ctx, cs, s.azs, *dryRun); err != nil {
		return true, errors.Wrap(err, "cannot sync cluster config")
	}

	logrus.Print("Sync process complete")
	return true, nil
}

func main() {
	flag.Parse()
	logrus.SetLevel(log.SanitizeLogLevel(*logLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logrus.Printf("sync pod starting, git commit %s", gitCommit)

	s := new(sync)
	ctx := context.Background()

	if err := s.init(ctx); err != nil {
		logrus.Fatalf("Cannot initialize sync: %v", err)
	}

	for {
		gotBlob, err := s.sync(ctx)
		if !gotBlob {
			// If we didn't manage to access the blob, error out and start
			// again.
			logrus.Fatalf("Error while accessing config blob: %v", err)
		}
		if err != nil {
			logrus.Printf("Error while syncing: %v", err)
		}
		if *once {
			return
		}
		<-time.After(*interval)
	}
}
