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

func sync() error {
	logrus.Print("Sync process started")

	ctx := context.Background()

	b, err := ioutil.ReadFile("_data/_out/azure.conf")
	if err != nil {
		return errors.Wrap(err, "cannot read _data/_out/azure.conf")
	}

	var m map[string]string
	if err := yaml.Unmarshal(b, &m); err != nil {
		return errors.Wrap(err, "cannot unmarshal _data/_out/azure.conf")
	}

	az, err := newAzureClients(ctx, m)
	if err != nil {
		return err
	}

	bsc := az.storage.GetBlobService()

	c := bsc.GetContainerReference("config")

	blob := c.GetBlobReference("config")

	logrus.Print("reading config blob")
	var rc io.ReadCloser
	err = wait.PollImmediateInfinite(time.Second, func() (bool, error) {
		rc, err = blob.Get(nil)

		if err, ok := err.(azstorage.AzureStorageServiceError); ok && err.StatusCode == http.StatusNotFound {
			return false, nil
		}

		return err == nil, err
	})
	if err != nil {
		return err
	}
	defer rc.Close()
	logrus.Print("read config blob")

	b, err = ioutil.ReadAll(rc)
	if err != nil {
		return err
	}

	var cs *api.OpenShiftManagedCluster
	if err := yaml.Unmarshal(b, &cs); err != nil {
		return err
	}

	if errs := api.Validate(cs, nil, false); len(errs) > 0 {
		return errors.Wrap(kerrors.NewAggregate(errs), "cannot validate _data/manifest.yaml")
	}

	if err := addons.Main(ctx, cs, az.accounts, *dryRun); err != nil {
		return errors.Wrap(err, "cannot sync cluster config")
	}

	logrus.Print("Sync process complete")
	return nil
}

func main() {
	flag.Parse()
	logrus.SetLevel(log.SanitizeLogLevel(*logLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logrus.Printf("sync pod starting, git commit %s", gitCommit)
	for {
		if err := sync(); err != nil {
			logrus.Printf("Error while syncing: %v", err)
		}
		if *once {
			return
		}
		<-time.After(*interval)
	}
}
