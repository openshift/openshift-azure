package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	azgraphrbac "github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"

	v20190430 "github.com/openshift/openshift-azure/pkg/api/2019-04-30"
	v20190930preview "github.com/openshift/openshift-azure/pkg/api/2019-09-30-preview"
	v20191027preview "github.com/openshift/openshift-azure/pkg/api/2019-10-27-preview"
	"github.com/openshift/openshift-azure/pkg/api/admin"
	fakerp "github.com/openshift/openshift-azure/pkg/fakerp/client"
	"github.com/openshift/openshift-azure/pkg/fakerp/shared"
	"github.com/openshift/openshift-azure/pkg/util/aadapp"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/graphrbac"
	v20190430client "github.com/openshift/openshift-azure/pkg/util/azureclient/openshiftmanagedcluster/2019-04-30"
	v20190930previewclient "github.com/openshift/openshift-azure/pkg/util/azureclient/openshiftmanagedcluster/2019-09-30-preview"
	v20191027previewclient "github.com/openshift/openshift-azure/pkg/util/azureclient/openshiftmanagedcluster/2019-10-27-preview"
	adminclient "github.com/openshift/openshift-azure/pkg/util/azureclient/openshiftmanagedcluster/admin"
)

var (
	method = flag.String("request", http.MethodPut, "Specify request to send to the OpenShift resource provider. Supported methods are PUT and DELETE.")

	adminManifest   = flag.String("admin-manifest", "", "If set, use the admin API to send this request.")
	restoreFromBlob = flag.String("restore-from-blob", "", "If set, request a restore of the cluster from the provided blob name.")
	apiVersion      = flag.String("api-version", "", "If set, request will be made with specific api. Defaults to latest")
)

const (
	defaultAadAppUri = "http://localhost/"
)

type Client struct {
	log       *logrus.Entry
	ac        *adminclient.Client
	rpcs      map[string]interface{}
	conf      *fakerp.Config
	rpURI     string
	aadClient graphrbac.ApplicationsClient
}

func validate() error {
	m := strings.ToUpper(*method)
	switch m {
	case http.MethodPut, http.MethodDelete:
	default:
		return fmt.Errorf("invalid request: %s, Supported methods are PUT and DELETE", strings.ToUpper(*method))
	}
	if *restoreFromBlob != "" && m == http.MethodDelete {
		return errors.New("cannot restore a cluster while requesting a DELETE")
	}
	return nil
}

func delete(ctx context.Context, log *logrus.Entry, rpc v20191027previewclient.OpenShiftManagedClustersClient, resourceGroup string, noWait bool) error {
	log.Info("deleting cluster")
	future, err := rpc.Delete(ctx, resourceGroup, resourceGroup)
	if err != nil {
		return err
	}
	if noWait {
		log.Info("will not wait for cluster deletion")
	} else {
		log.Info("waiting for cluster deletion")
		if err := future.WaitForCompletionRef(ctx, rpc.Client); err != nil {
			return err
		}
		log.Info("deleted cluster")
	}
	return nil
}

func (c Client) createOrUpdatev20191027preview(ctx context.Context) (*v20191027preview.OpenShiftManagedCluster, error) {
	var oc *v20191027preview.OpenShiftManagedCluster
	err := fakerp.GenerateManifest(c.conf, &oc)
	if err != nil {
		return nil, fmt.Errorf("failed reading manifest: %v", err)
	}
	c.log.Info("creating/updating cluster")
	resp, err := c.rpcs["2019-10-27-preview"].(v20191027previewclient.OpenShiftManagedClustersClient).CreateOrUpdateAndWait(ctx, c.conf.ResourceGroup, c.conf.ResourceGroup, *oc)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response: %s", resp.Status)
	}
	err = fakerp.WriteClusterConfigToManifest(oc, "_data/manifest.yaml")
	if err != nil {
		return nil, err
	}

	c.log.Info("created/updated cluster")
	return &resp, nil
}

func (c Client) createOrUpdatev20190930preview(ctx context.Context) (*v20190930preview.OpenShiftManagedCluster, error) {
	var oc *v20190930preview.OpenShiftManagedCluster
	err := fakerp.GenerateManifest(c.conf, &oc)
	if err != nil {
		return nil, fmt.Errorf("failed reading manifest: %v", err)
	}
	c.log.Info("creating/updating cluster")
	resp, err := c.rpcs["2019-09-30-preview"].(v20190930previewclient.OpenShiftManagedClustersClient).CreateOrUpdateAndWait(ctx, c.conf.ResourceGroup, c.conf.ResourceGroup, *oc)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response: %s", resp.Status)
	}
	err = fakerp.WriteClusterConfigToManifest(oc, "_data/manifest.yaml")
	if err != nil {
		return nil, err
	}

	c.log.Info("created/updated cluster")
	return &resp, nil
}

func (c Client) createOrUpdatev20190430(ctx context.Context) (*v20190430.OpenShiftManagedCluster, error) {
	var oc *v20190430.OpenShiftManagedCluster
	err := fakerp.GenerateManifest(c.conf, &oc)
	if err != nil {
		return nil, fmt.Errorf("failed reading manifest: %v", err)
	}
	c.log.Info("creating/updating cluster")
	resp, err := c.rpcs["2019-10-27-preview"].(v20190430client.OpenShiftManagedClustersClient).CreateOrUpdateAndWait(ctx, c.conf.ResourceGroup, c.conf.ResourceGroup, *oc)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response: %s", resp.Status)
	}
	err = fakerp.WriteClusterConfigToManifest(oc, "_data/manifest.yaml")
	if err != nil {
		return nil, err
	}

	c.log.Info("created/updated cluster")
	return &resp, nil
}

func (c Client) createOrUpdateAdmin(ctx context.Context) (*v20191027preview.OpenShiftManagedCluster, error) {
	var oc *admin.OpenShiftManagedCluster
	c.conf.Manifest = *adminManifest
	err := fakerp.GenerateManifest(c.conf, &oc)
	if err != nil {
		return nil, fmt.Errorf("failed reading admin manifest: %v", err)
	}
	c.log.Info("creating/updating cluster")
	resonse, err := c.ac.CreateOrUpdate(ctx, c.conf.ResourceGroup, c.conf.ResourceGroup, oc)
	if err != nil {
		return nil, err
	}
	c.log.Info("created/updated cluster")
	err = fakerp.WriteClusterConfigToManifest(resonse, "_data/manifest.yaml")
	if err != nil {
		return nil, err
	}
	cluster, err := c.rpcs["2019-10-27-preview"].(v20191027previewclient.OpenShiftManagedClustersClient).Get(ctx, c.conf.ResourceGroup, c.conf.ResourceGroup)
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

func main() {
	flag.Parse()
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log := logrus.NewEntry(logrus.StandardLogger())
	ctx := context.Background()

	if err := validate(); err != nil {
		log.Fatal(err)
	}

	isDelete := strings.ToUpper(*method) == http.MethodDelete
	conf, err := fakerp.NewClientConfig(log)
	if err != nil {
		log.Fatal(err)
	}

	if !isDelete {
		log.Infof("ensuring resource group %s", conf.ResourceGroup)
		err = fakerp.EnsureResourceGroup(log, conf)
		if err != nil {
			log.Fatal(err)
		}
	}

	rpURI := fmt.Sprintf("http://%s", shared.LocalHttpAddr)

	authorizer, err := azureclient.NewAuthorizer(conf.ClientID, conf.ClientSecret, conf.TenantID, "")
	if err != nil {
		log.Fatal(err)
	}
	clientv20191027preview := v20191027previewclient.NewOpenShiftManagedClustersClientWithBaseURI(rpURI, conf.SubscriptionID)
	clientv20191027preview.Authorizer = authorizer
	clientv201909307preview := v20190930previewclient.NewOpenShiftManagedClustersClientWithBaseURI(rpURI, conf.SubscriptionID)
	clientv201909307preview.Authorizer = authorizer
	clientv20190430 := v20190430client.NewOpenShiftManagedClustersClientWithBaseURI(rpURI, conf.SubscriptionID)
	clientv20190430.Authorizer = authorizer

	clients := map[string]interface{}{
		"2019-10-27-preview": clientv20191027preview,
		"2019-09-30-preview": clientv201909307preview,
		"2019-04-30":         clientv20190430,
	}

	// setup the aad clients
	graphauthorizer, err := azureclient.NewAuthorizerFromEnvironment(azure.PublicCloud.GraphEndpoint)
	if err != nil {
		log.Fatal(fmt.Errorf("cannot get authorizer: %v", err))
	}
	aadClient := graphrbac.NewApplicationsClient(ctx, log, conf.TenantID, graphauthorizer)

	client := Client{
		log:       log,
		rpURI:     rpURI,
		conf:      conf,
		rpcs:      clients,
		ac:        adminclient.NewClient(rpURI, conf.SubscriptionID),
		aadClient: aadClient,
	}

	if isDelete {
		client.Delete(ctx)
	}

	if *restoreFromBlob != "" {
		err = client.ac.Restore(ctx, conf.ResourceGroup, conf.ResourceGroup, *restoreFromBlob)
		if err != nil {
			log.Fatal(err)
		}
	}

	url, err := client.Create(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nCluster available at https://%s/\n", url)
}

func (c Client) Delete(ctx context.Context) {
	aadObjID, err := aadapp.GetApplicationObjectIDFromAppID(ctx, c.aadClient, c.conf.AADClientID)
	if err != nil {
		log.Fatal(err)
	}
	aadApp, err := c.aadClient.Get(ctx, aadObjID)
	if err != nil {
		log.Fatal(err)
	}
	err = delete(ctx, c.log, c.rpcs["2019-10-27-preview"].(v20191027previewclient.OpenShiftManagedClustersClient), c.conf.ResourceGroup, c.conf.NoWait)
	if err != nil {
		log.Fatal(err)
	}
	// unlink cluster from aad app
	if err := unlinkClusterFromAadApp(ctx, c.log, &c.aadClient, &aadApp, c.conf); err != nil {
		log.Fatal(err)
	}
	return
}

func (c Client) Create(ctx context.Context) (string, error) {
	c.log.Info("create()")
	if *adminManifest != "" {
		_, err := c.createOrUpdateAdmin(ctx)
		if err != nil {
			return "", err
		}
		return "", nil
	}

	// If no MANIFEST has been provided and this is a cluster
	// creation, default to the test manifest.
	if !shared.IsUpdate() && c.conf.Manifest == "" {
		c.conf.Manifest = "test/manifests/fakerp/create.yaml"
	} else {
		// If this is a cluster upgrade, reuse the existing manifest.
		c.conf.Manifest = "_data/manifest.yaml"
	}

	var consoleURL string
	switch *apiVersion {
	case "2019-04-30":
		oc, err := c.createOrUpdatev20190430(ctx)
		if err != nil {
			return "", err
		}
		consoleURL = *oc.Properties.PublicHostname
	case "2019-09-30-preview":
		oc, err := c.createOrUpdatev20190930preview(ctx)
		if err != nil {
			return "", err
		}
		consoleURL = *oc.Properties.PublicHostname
	// case "2019-09-30-preview" forward to default
	default:
		oc, err := c.createOrUpdatev20190930preview(ctx)
		if err != nil {
			return "", err
		}
		consoleURL = *oc.Properties.PublicHostname
	}

	aadObjID, err := aadapp.GetApplicationObjectIDFromAppID(ctx, c.aadClient, c.conf.AADClientID)
	if err != nil {
		log.Fatal(err)
	}
	aadApp, err := c.aadClient.Get(ctx, aadObjID)
	if err != nil {
		log.Fatal(err)
	}

	if err := linkClusterToAadApp(ctx, c.log, &c.aadClient, &aadApp, c.conf); err != nil {
		return "", err
	}
	return consoleURL, nil

}

func linkClusterToAadApp(ctx context.Context, log *logrus.Entry, aadClient *graphrbac.ApplicationsClient, aadApp *azgraphrbac.Application, conf *fakerp.Config) error {
	if len(conf.AADClientID) > 0 && conf.AADClientID != conf.ClientID {
		callbackURL := fmt.Sprintf("https://openshift.%s.osadev.cloud/oauth2callback/Azure%%20AD", conf.ResourceGroup)
		links := addUrl(*aadApp.ReplyUrls, callbackURL)
		links = removeUrl(links, defaultAadAppUri)
		err := aadapp.UpdateAADApp(ctx, *aadClient, *aadApp.ObjectID, callbackURL, links)
		if err != nil {
			return fmt.Errorf("could not update aad app: %v", err)
		}
		log.Infof("linked cluster %s to aad object id %s", conf.ResourceGroup, *aadApp.ObjectID)
	}
	return nil
}

func unlinkClusterFromAadApp(ctx context.Context, log *logrus.Entry, aadClient *graphrbac.ApplicationsClient, aadApp *azgraphrbac.Application, conf *fakerp.Config) error {
	if len(conf.AADClientID) > 0 && conf.AADClientID != conf.ClientID {
		callbackURL := fmt.Sprintf("https://openshift.%s.osadev.cloud/oauth2callback/Azure%%20AD", conf.ResourceGroup)
		links := removeUrl(*aadApp.ReplyUrls, callbackURL)
		if len(links) == 0 {
			links = append(links, defaultAadAppUri)
		}
		err := aadapp.UpdateAADApp(ctx, *aadClient, *aadApp.ObjectID, callbackURL, links)
		if err != nil {
			return fmt.Errorf("could not update aad app: %v", err)
		}
		log.Infof("unlinked cluster %s from aad object id %s", conf.ResourceGroup, *aadApp.ObjectID)
	}
	return nil
}

func addUrl(urls []string, newUrl string) []string {
	for _, url := range urls {
		if url == newUrl {
			return urls
		}
	}
	urls = append(urls, newUrl)
	return urls
}

func removeUrl(urls []string, deleteUrl string) []string {
	var newUrls []string
	for _, url := range urls {
		if url != deleteUrl {
			newUrls = append(newUrls, url)
		}
	}
	return newUrls
}
