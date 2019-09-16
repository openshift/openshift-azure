package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"

	azgraphrbac "github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	v20190930preview "github.com/openshift/openshift-azure/pkg/api/2019-09-30-preview"
	"github.com/openshift/openshift-azure/pkg/api/admin"
	fakerp "github.com/openshift/openshift-azure/pkg/fakerp/client"
	"github.com/openshift/openshift-azure/pkg/fakerp/shared"
	"github.com/openshift/openshift-azure/pkg/util/aadapp"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/graphrbac"
	v20190930previewclient "github.com/openshift/openshift-azure/pkg/util/azureclient/openshiftmanagedcluster/2019-09-30-preview"
	adminclient "github.com/openshift/openshift-azure/pkg/util/azureclient/openshiftmanagedcluster/admin"
	utilerrors "github.com/openshift/openshift-azure/pkg/util/errors"
)

var (
	method  = flag.String("request", http.MethodPut, "Specify request to send to the OpenShift resource provider. Supported methods are PUT and DELETE.")
	useProd = flag.Bool("use-prod", false, "If true, send the request to the production OpenShift resource provider.")

	adminManifest   = flag.String("admin-manifest", "", "If set, use the admin API to send this request.")
	restoreFromBlob = flag.String("restore-from-blob", "", "If set, request a restore of the cluster from the provided blob name.")
)

func validate() error {
	m := strings.ToUpper(*method)
	switch m {
	case http.MethodPut, http.MethodDelete:
	default:
		return fmt.Errorf("invalid request: %s, Supported methods are PUT and DELETE", strings.ToUpper(*method))
	}
	if *adminManifest != "" && *useProd {
		return errors.New("sending requests to the Admin API is not supported yet in the production RP")
	}
	if *restoreFromBlob != "" && *useProd {
		return errors.New("restoring clusters is not supported yet in the production RP")
	}
	if *restoreFromBlob != "" && m == http.MethodDelete {
		return errors.New("cannot restore a cluster while requesting a DELETE")
	}
	return nil
}

func delete(ctx context.Context, log *logrus.Entry, rpc v20190930previewclient.OpenShiftManagedClustersClient, resourceGroup string, noWait bool) error {
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

func createOrUpdatev20190930preview(ctx context.Context, log *logrus.Entry, rpc v20190930previewclient.OpenShiftManagedClustersClient, resourceGroup string, oc *v20190930preview.OpenShiftManagedCluster, manifestFile string) (*v20190930preview.OpenShiftManagedCluster, error) {
	log.Info("creating/updating cluster")
	resp, err := rpc.CreateOrUpdateAndWait(ctx, resourceGroup, resourceGroup, *oc)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response: %s", resp.Status)
	}
	log.Info("created/updated cluster")
	return &resp, nil
}

func createOrUpdateAdmin(ctx context.Context, log *logrus.Entry, ac *adminclient.Client, rpc v20190930previewclient.OpenShiftManagedClustersClient, resourceGroup string, oc *admin.OpenShiftManagedCluster, manifestFile string) (*v20190930preview.OpenShiftManagedCluster, error) {
	log.Info("creating/updating cluster")
	resp, err := ac.CreateOrUpdate(ctx, resourceGroup, resourceGroup, oc)
	if err != nil {
		return nil, err
	}
	data, err := yaml.Marshal(resp)
	if err != nil {
		return nil, err
	}
	log.Info("created/updated cluster")
	err = ioutil.WriteFile(manifestFile, data, 0600)
	if err != nil {
		return nil, err
	}
	cluster, err := rpc.Get(ctx, resourceGroup, resourceGroup)
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

func execute(
	ctx context.Context,
	log *logrus.Entry,
	ac *adminclient.Client,
	rpc v20190930previewclient.OpenShiftManagedClustersClient,
	conf *fakerp.Config,
	adminManifest string,
) (*v20190930preview.OpenShiftManagedCluster, error) {
	if adminManifest != "" {
		var oc *admin.OpenShiftManagedCluster
		err := fakerp.GenerateManifest(conf, adminManifest, &oc)
		if err != nil {
			return nil, fmt.Errorf("failed reading admin manifest: %v", err)
		}
		defaultAdminManifest := "_data/manifest-admin.yaml"
		return createOrUpdateAdmin(ctx, log, ac, rpc, conf.ResourceGroup, oc, defaultAdminManifest)
	}

	defaultManifestFile := "_data/manifest.yaml"
	// TODO: Configuring this is probably not needed
	manifest := conf.Manifest
	// If no MANIFEST has been provided and this is a cluster
	// creation, default to the test manifest.
	if !shared.IsUpdate() && manifest == "" {
		if *useProd {
			manifest = "test/manifests/realrp/create.yaml"
		} else {
			manifest = "test/manifests/fakerp/create.yaml"
		}
	}

	// If this is a cluster upgrade, reuse the existing manifest.
	if manifest == "" {
		manifest = defaultManifestFile
	}

	var oc *v20190930preview.OpenShiftManagedCluster
	err := fakerp.GenerateManifest(conf, manifest, &oc)
	if err != nil {
		return nil, fmt.Errorf("failed reading manifest: %v", err)
	}

	oc, err = createOrUpdatev20190930preview(ctx, log, rpc, conf.ResourceGroup, oc, defaultManifestFile)
	if err != nil {
		return nil, err
	}

	err = fakerp.WriteClusterConfigToManifest(oc, defaultManifestFile)
	if err != nil {
		return nil, err
	}

	return oc, nil
}

func linkClusterToAadApp(ctx context.Context, log *logrus.Entry, aadClient *graphrbac.ApplicationsClient, aadApp *azgraphrbac.Application, conf *fakerp.Config) error {
	if len(conf.AADClientID) > 0 && conf.AADClientID != conf.ClientID {
		log.Infof("linked cluster %s to aad object id %s", conf.ResourceGroup, *aadApp.ObjectID)
		callbackURL := fmt.Sprintf("https://openshift.%s.osadev.cloud/oauth2callback/Azure%%20AD", conf.ResourceGroup)
		addUrl := func(urls []string, newUrl string) []string {
			for _, url := range urls {
				if url == newUrl {
					return urls
				}
			}
			urls = append(urls, newUrl)
			return urls
		}
		links := addUrl(*aadApp.ReplyUrls, callbackURL)
		err := aadapp.UpdateAADApp(ctx, *aadClient, *aadApp.ObjectID, links)
		if err != nil {
			return fmt.Errorf("could not update aad app: %v", err)
		}
	}
	return nil
}

func unlinkClusterFromAadApp(ctx context.Context, log *logrus.Entry, aadClient *graphrbac.ApplicationsClient, aadApp *azgraphrbac.Application, conf *fakerp.Config) error {
	if len(conf.AADClientID) > 0 && conf.AADClientID != conf.ClientID {
		log.Infof("unlinked cluster %s from aad object id %s", conf.ResourceGroup, *aadApp.ObjectID)
		callbackURL := fmt.Sprintf("https://openshift.%s.osadev.cloud/oauth2callback/Azure%%20AD", conf.ResourceGroup)
		removeUrl := func(urls []string, deleteUrl string) []string {
			var newUrls []string
			for _, url := range urls {
				if url != deleteUrl {
					newUrls = append(newUrls, url)
				}
			}
			return newUrls
		}
		links := removeUrl(*aadApp.ReplyUrls, callbackURL)
		err := aadapp.UpdateAADApp(ctx, *aadClient, *aadApp.ObjectID, links)
		if err != nil {
			return fmt.Errorf("could not update aad app: %v", err)
		}
	}
	return nil
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
	conf, err := fakerp.NewConfig(log)
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

	// simulate the RP
	rpURL := v20190930previewclient.DefaultBaseURI
	if !*useProd {
		rpURL = fmt.Sprintf("http://%s", shared.LocalHttpAddr)

		// wait for the fake RP to start
		err := wait.PollImmediate(time.Second, time.Minute, func() (bool, error) {
			c, err := net.Dial("tcp", shared.LocalHttpAddr)
			if utilerrors.IsMatchingSyscallError(err, syscall.ECONNREFUSED) {
				return false, nil
			}
			if err != nil {
				return false, err
			}
			return true, c.Close()
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	// setup the osa clients
	adminClient := adminclient.NewClient(rpURL, conf.SubscriptionID)
	v20190930previewClient := v20190930previewclient.NewOpenShiftManagedClustersClientWithBaseURI(rpURL, conf.SubscriptionID)
	authorizer, err := azureclient.NewAuthorizer(conf.ClientID, conf.ClientSecret, conf.TenantID, "")
	if err != nil {
		log.Fatal(err)
	}
	v20190930previewClient.Authorizer = authorizer

	// setup the aad clients
	graphauthorizer, err := azureclient.NewAuthorizerFromEnvironment(azure.PublicCloud.GraphEndpoint)
	if err != nil {
		log.Fatal(fmt.Errorf("cannot get authorizer: %v", err))
	}
	aadClient := graphrbac.NewApplicationsClient(ctx, log, conf.TenantID, graphauthorizer)
	aadObjID, err := aadapp.GetApplicationObjectIDFromAppID(ctx, aadClient, conf.AADClientID)
	if err != nil {
		log.Fatal(err)
	}
	aadApp, err := aadClient.Get(ctx, aadObjID)
	if err != nil {
		log.Fatal(err)
	}

	if isDelete {
		err = delete(ctx, log, v20190930previewClient, conf.ResourceGroup, conf.NoWait)
		if err != nil {
			log.Fatal(err)
		}
		// unlink cluster from aad app
		if err := unlinkClusterFromAadApp(ctx, log, &aadClient, &aadApp, conf); err != nil {
			log.Fatal(err)
		}
		return
	}

	if *restoreFromBlob != "" {
		err = adminClient.Restore(ctx, conf.ResourceGroup, conf.ResourceGroup, *restoreFromBlob)
		if err != nil {
			log.Fatal(err)
		}
	}

	oc, err := execute(ctx, log, adminClient, v20190930previewClient, conf, *adminManifest)
	if err != nil {
		log.Fatal(err)
	}

	if err := linkClusterToAadApp(ctx, log, &aadClient, &aadApp, conf); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nCluster available at https://%s/\n", *oc.Properties.PublicHostname)
}
