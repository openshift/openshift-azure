package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	azureclientstorage "github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
	"github.com/openshift/openshift-azure/pkg/util/nodeconf"
)

var (
	logLevel    = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")
	interval    = flag.Duration("interval", 5*time.Second, "How often to retry the download.")
	blobName    = flag.String("blobname", "", "name of the blog to download")
	destination = flag.String("destination", "", "where to place the blob on the filesystem")
	gitCommit   = "unknown"
)

type getBackup struct {
	bsc azureclientstorage.BlobStorageClient
}

func getBlobStorageClient(ctx context.Context) (azureclientstorage.BlobStorageClient, error) {
	m, err := nodeconf.GetAzureConf()
	if err != nil {
		return nil, err
	}
	authorizer, err := azureclient.NewAuthorizer(m["aadClientId"], m["aadClientSecret"], m["tenantId"])
	if err != nil {
		return nil, err
	}
	accounts := azureclient.NewAccountsClient(m["subscriptionId"], authorizer, nil)
	accts, err := accounts.ListByResourceGroup(ctx, m["resourceGroup"])
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

	keys, err := accounts.ListKeys(ctx, m["resourceGroup"], *acct.Name)
	if err != nil {
		return nil, err
	}

	storage, err := azureclientstorage.NewClient(*acct.Name, *(*keys.Keys)[0].Value,
		azureclientstorage.DefaultBaseURL, azureclientstorage.DefaultAPIVersion, true)
	if err != nil {
		return nil, err
	}
	return storage.GetBlobService(), nil
}

func parseBucketAndKey(path string) (string, string, error) {
	toks := strings.SplitN(path, "/", 2)
	if len(toks) != 2 || len(toks[0]) == 0 || len(toks[1]) == 0 {
		return "", "", fmt.Errorf("Invalid ABS path (%v)", path)
	}
	return toks[0], toks[1], nil
}

func (r *getBackup) open(path string) (io.ReadCloser, error) {
	container, key, err := parseBucketAndKey(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse abs container and key: %v", err)
	}

	containerRef := r.bsc.GetContainerReference(container)
	containerExists, err := containerRef.Exists()
	if err != nil {
		return nil, err
	}

	if !containerExists {
		return nil, fmt.Errorf("container %v does not exist", container)
	}

	blob := containerRef.GetBlobReference(key)
	return blob.Get(&azstorage.GetBlobOptions{})
}

func (r *getBackup) copy(ctx context.Context, srcBlob, destPath string) error {
	logrus.Printf("copy blob %v to filesystem %v", srcBlob, destPath)
	var rc io.ReadCloser
	var err error
	err = wait.PollImmediateInfinite(time.Second, func() (bool, error) {
		rc, err = r.open(srcBlob)
		if err, ok := err.(azstorage.AzureStorageServiceError); ok && err.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return err == nil, err
	})
	if err != nil {
		return err
	}
	defer rc.Close()
	logrus.Print("read blob")

	b, err := ioutil.ReadAll(rc)
	if err != nil {
		return err
	}
	logrus.Print("creating ", destPath)
	df, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer df.Close()
	_, err = df.Write(b)
	logrus.Print("Write returned ", err)
	return err
}

func main() {
	flag.Parse()
	logrus.SetLevel(log.SanitizeLogLevel(*logLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logrus.Printf("getbackup pod starting, git commit %s", gitCommit)

	r := new(getBackup)
	ctx := context.Background()

	var err error
	r.bsc, err = getBlobStorageClient(ctx)
	if err != nil {
		logrus.Fatalf("Cannot get storage account getBackup: %v", err)
	}

	for i := 1; i <= 10; i++ {
		err = r.copy(ctx, *blobName, *destination)
		if err != nil {
			logrus.Fatalf("Error while getting az blob: %v->%v %v", *blobName, *destination, err)
			<-time.After(*interval)
		} else {
			os.Exit(0)
		}
	}
	logrus.Fatalf("tried 10 times to copy backup file - still failed")
}
