package main

import (
	"context"
	"flag"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

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
)

var (
	dryRun    = flag.Bool("dry-run", false, "Print resources to be synced instead of mutating cluster state.")
	once      = flag.Bool("run-once", false, "If true, run only once then quit.")
	interval  = flag.Duration("interval", 3*time.Minute, "How often the sync process going to be rerun.")
	logLevel  = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")
	gitCommit = "unknown"
)

func sync() error {
	logrus.Print("Sync process started")
	ctx := context.Background()

	var config = api.PluginConfig{SyncImage: os.Getenv("SYNC_IMAGE"),
		AcceptLanguages: []string{"en-us"}}

	b, err := ioutil.ReadFile("_data/_out/azure.conf")
	if err != nil {
		return errors.Wrap(err, "cannot read _data/_out/azure.conf")
	}

	var m map[string]string
	if err := yaml.Unmarshal(b, &m); err != nil {
		return errors.Wrap(err, "cannot unmarshal _data/_out/azure.conf")
	}

	authorizer, err := azureclient.NewAuthorizer(m["aadClientId"], m["aadClientSecret"], m["tenantId"], m["subscriptionId"])
	if err != nil {
		return err
	}
	accountClient := azureclient.NewAccountsClient(m["subscriptionId"], authorizer, config)
	storageAcc, err := accountClient.GetStorageAccount(ctx, m["resourceGroup"], "config")
	if err != nil {
		return err
	}
	storageClient, err := azureclient.NewStorageClient(storageAcc["name"], storageAcc["key"])
	if err != nil {
		return err
	}

	c := storageClient.GetContainerReference("config")
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

	if err := addons.Main(cs, accountClient, *dryRun); err != nil {
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
