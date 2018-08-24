package main

import (
	"context"
	"flag"
	"io/ioutil"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/addons"
	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/validate"
)

var (
	dryRun   = flag.Bool("dry-run", false, "Print resources to be synced instead of mutating cluster state.")
	once     = flag.Bool("run-once", false, "If true, run only once then quit.")
	interval = flag.Duration("interval", 3*time.Minute, "How often the sync process going to be rerun.")
	logLevel = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")
)

type azureStorageClient struct {
	accounts storage.AccountsClient
	storage  azstorage.Client
}

func newAzureClients(ctx context.Context, m map[string]string) (*azureStorageClient, error) {
	config := auth.NewClientCredentialsConfig(m["aadClientId"], m["aadClientSecret"], m["tenantId"])
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, err
	}

	clients := &azureStorageClient{}
	clients.accounts = storage.NewAccountsClient(m["subscriptionId"])
	clients.accounts.Authorizer = authorizer

	accts, err := clients.accounts.ListByResourceGroup(context.Background(), m["resourceGroup"])
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

	keys, err := clients.accounts.ListKeys(context.Background(), m["resourceGroup"], *acct.Name)
	if err != nil {
		return nil, err
	}

	clients.storage, err = azstorage.NewClient(*acct.Name, *(*keys.Keys)[0].Value, azstorage.DefaultBaseURL, azstorage.DefaultAPIVersion, true)
	if err != nil {
		return nil, err
	}

	return clients, nil
}

func sync() error {
	logrus.Print("Sync process started")

	b, err := ioutil.ReadFile("_data/_out/azure.conf")
	if err != nil {
		return errors.Wrap(err, "cannot read _data/_out/azure.conf")
	}

	var m map[string]string
	if err := yaml.Unmarshal(b, &m); err != nil {
		return errors.Wrap(err, "cannot unmarshal _data/_out/azure.conf")
	}

	az, err := newAzureClients(context.Background(), m)
	if err != nil {
		return err
	}

	bsc := az.storage.GetBlobService()

	c := bsc.GetContainerReference("config")

	blob := c.GetBlobReference("config")

	rc, err := blob.Get(nil)
	if err != nil {
		return err
	}
	defer rc.Close()

	b, err = ioutil.ReadAll(rc)
	if err != nil {
		return err
	}

	var cs *acsapi.OpenShiftManagedCluster
	if err := yaml.Unmarshal(b, &cs); err != nil {
		return err
	}

	if errs := validate.Validate(cs, nil, false); len(errs) > 0 {
		return errors.Wrap(kerrors.NewAggregate(errs), "cannot validate _data/manifest.yaml")
	}

	if err := addons.Main(cs, *dryRun); err != nil {
		return errors.Wrap(err, "cannot sync cluster config")
	}

	logrus.Print("Sync process complete")
	return nil
}

func main() {
	flag.Parse()
	logrus.SetLevel(log.SanitizeLogLevel(*logLevel))

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
