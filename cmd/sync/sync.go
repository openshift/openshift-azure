package main

import (
	"context"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/addons"
	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	azureclientstorage "github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

var (
	dryRun    = flag.Bool("dry-run", false, "Print resources to be synced instead of mutating cluster state.")
	once      = flag.Bool("run-once", false, "If true, run only once then quit.")
	interval  = flag.Duration("interval", 3*time.Minute, "How often the sync process going to be rerun.")
	logLevel  = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")
	gitCommit = "unknown"
)

type azureStorageClient struct {
	accounts azureclient.AccountsClient
	storage  azureclientstorage.Client
}

func newAzureClients(ctx context.Context, m map[string]string) (*azureStorageClient, error) {
	authorizer, err := azureclient.NewAuthorizer(m["aadClientId"], m["aadClientSecret"], m["tenantId"])
	if err != nil {
		return nil, err
	}

	clients := &azureStorageClient{}
	clients.accounts = azureclient.NewAccountsClient(m["subscriptionId"], authorizer, nil)

	accts, err := clients.accounts.ListByResourceGroup(ctx, m["resourceGroup"])
	if err != nil {
		return nil, err
	}

	var acct storage.Account
	var found bool
	for _, acct = range *accts.Value {
		found = acct.Tags["type"] != nil && *acct.Tags["type"] == "config"
		if found {
			break
		}
	}
	if !found {
		return nil, errors.New("storage account not found")
	}
	logrus.Printf("found account %s", *acct.Name)

	keys, err := clients.accounts.ListKeys(ctx, m["resourceGroup"], *acct.Name)
	if err != nil {
		return nil, err
	}

	clients.storage, err = azureclientstorage.NewClient(*acct.Name, *(*keys.Keys)[0].Value, azureclientstorage.DefaultBaseURL, azureclientstorage.DefaultAPIVersion, true)
	if err != nil {
		return nil, err
	}

	return clients, nil
}

type sync struct {
	stc *azureStorageClient

	blob azureclientstorage.Blob
}

func (s *sync) init(ctx context.Context) error {
	b, err := ioutil.ReadFile("_data/_out/azure.conf")
	if err != nil {
		return errors.Wrap(err, "cannot read _data/_out/azure.conf")
	}

	var m map[string]string
	if err := yaml.Unmarshal(b, &m); err != nil {
		return errors.Wrap(err, "cannot unmarshal _data/_out/azure.conf")
	}

	s.stc, err = newAzureClients(ctx, m)
	if err != nil {
		return err
	}

	bsc := s.stc.storage.GetBlobService()
	c := bsc.GetContainerReference("config")
	s.blob = c.GetBlobReference("config")
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

	if err := addons.Main(ctx, cs, s.stc.accounts, *dryRun); err != nil {
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
