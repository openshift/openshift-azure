package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	admin "github.com/openshift/openshift-azure/pkg/api/admin/api"
	fakerp "github.com/openshift/openshift-azure/pkg/fakerp/client"
	"github.com/openshift/openshift-azure/pkg/fakerp/shared"
	"github.com/openshift/openshift-azure/pkg/util/aadapp"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	v20180930previewclient "github.com/openshift/openshift-azure/pkg/util/azureclient/openshiftmanagedcluster/2018-09-30-preview"
	adminclient "github.com/openshift/openshift-azure/pkg/util/azureclient/openshiftmanagedcluster/admin"
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
		return errors.New("cannot restore a cluster while requesting a DELETE?")
	}
	return nil
}

func delete(ctx context.Context, log *logrus.Entry, rpc v20180930previewclient.OpenShiftManagedClustersClient, resourceGroup string, noWait bool) error {
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

func createOrUpdatev20180930preview(ctx context.Context, log *logrus.Entry, rpc v20180930previewclient.OpenShiftManagedClustersClient, resourceGroup string, oc *v20180930preview.OpenShiftManagedCluster, manifestFile string) (*v20180930preview.OpenShiftManagedCluster, error) {
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

func createOrUpdateAdmin(ctx context.Context, log *logrus.Entry, rpc *adminclient.Client, resourceGroup string, oc *admin.OpenShiftManagedCluster, manifestFile string) error {
	log.Info("creating/updating cluster")
	if oc.Properties != nil {
		oc.Properties.ProvisioningState = nil
	}
	resp, err := rpc.CreateOrUpdate(ctx, resourceGroup, resourceGroup, oc)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response: %s", resp.Status)
	}
	data, err := yaml.Marshal(resp)
	if err != nil {
		return err
	}
	log.Info("created/updated cluster")
	return ioutil.WriteFile(manifestFile, data, 0600)
}

func execute(
	ctx context.Context,
	log *logrus.Entry,
	ac *adminclient.Client,
	rpc v20180930previewclient.OpenShiftManagedClustersClient,
	conf *fakerp.Config,
	adminManifest string,
) (*v20180930preview.OpenShiftManagedCluster, error) {
	if adminManifest != "" {
		oc, err := fakerp.GenerateManifestAdmin(adminManifest)
		if err != nil {
			return nil, fmt.Errorf("failed reading admin manifest: %v", err)
		}
		defaultAdminManifest := "_data/manifest-admin.yaml"
		return nil, createOrUpdateAdmin(ctx, log, ac, conf.ResourceGroup, oc, defaultAdminManifest)
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

	oc, err := fakerp.GenerateManifest(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed reading manifest: %v", err)
	}

	oc, err = createOrUpdatev20180930preview(ctx, log, rpc, conf.ResourceGroup, oc, defaultManifestFile)
	if err != nil {
		return nil, err
	}

	err = fakerp.WriteClusterConfigToManifest(oc, defaultManifestFile)
	if err != nil {
		return nil, err
	}

	return oc, nil
}

func updateAadApplication(ctx context.Context, oc *v20180930preview.OpenShiftManagedCluster, log *logrus.Entry, conf *fakerp.Config) error {
	if len(conf.AADClientID) > 0 && conf.AADClientID != conf.ClientID {
		log.Info("updating the aad application")
		graphauthorizer, err := azureclient.NewAuthorizerFromEnvironment(azure.PublicCloud.GraphEndpoint)
		if err != nil {
			return fmt.Errorf("cannot get authorizer: %v", err)
		}

		aadClient := azureclient.NewRBACApplicationsClient(ctx, conf.TenantID, graphauthorizer)
		objID, err := aadapp.GetApplicationObjectIDFromAppID(ctx, aadClient, conf.AADClientID)
		if err != nil {
			return err
		}

		callbackURL := fmt.Sprintf("https://%s/oauth2callback/Azure%%20AD", *oc.Properties.PublicHostname)
		err = aadapp.UpdateAADApp(ctx, aadClient, objID, callbackURL)
		if err != nil {
			return fmt.Errorf("cannot update aad app secret: %v", err)
		}
	}

	return nil
}

func isConnectionRefused(err error) bool {
	if autoRestErr, ok := err.(autorest.DetailedError); ok {
		if urlErr, ok := autoRestErr.Original.(*url.Error); ok {
			if netErr, ok := urlErr.Err.(*net.OpError); ok {
				if sysErr, ok := netErr.Err.(*os.SyscallError); ok {
					if sysErr.Err == syscall.ECONNREFUSED {
						return true
					}
				}
			}
		}
	}
	return false
}

func main() {
	flag.Parse()
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log := logrus.NewEntry(logrus.StandardLogger())

	if err := validate(); err != nil {
		log.Fatal(err)
	}

	isDelete := strings.ToUpper(*method) == http.MethodDelete
	isUpdate := shared.IsUpdate()
	conf, err := fakerp.NewConfig(log, !isDelete)
	if err != nil {
		log.Fatal(err)
	}

	if !isDelete {
		log.Infof("creating resource group %s", conf.ResourceGroup)
		if isCreate, err := fakerp.CreateResourceGroup(conf); err != nil {
			log.Fatal(err)
		} else if !isCreate {
			log.Infof("reusing existing resource group %s", conf.ResourceGroup)
		}
	}

	// simulate the RP
	rpURL := v20180930previewclient.DefaultBaseURI
	if !*useProd {
		rpURL = fmt.Sprintf("http://%s", shared.LocalHttpAddr)
	}

	// setup the osa clients
	adminClient := adminclient.NewClient(rpURL, conf.SubscriptionID)
	v20180930previewClient := v20180930previewclient.NewOpenShiftManagedClustersClientWithBaseURI(rpURL, conf.SubscriptionID)
	authorizer, err := azureclient.NewAuthorizer(conf.ClientID, conf.ClientSecret, conf.TenantID, "")
	if err != nil {
		log.Fatal(err)
	}
	v20180930previewClient.Authorizer = authorizer

	ctx := context.Background()
	if isDelete {
		if err := wait.PollImmediate(time.Second, 1*time.Hour, func() (bool, error) {
			if err := delete(ctx, log, v20180930previewClient, conf.ResourceGroup, conf.NoWait); err != nil {
				if isConnectionRefused(err) {
					return false, nil
				}
				return false, err
			}
			return true, nil
		}); err != nil {
			log.Fatal(err)
		}
		return
	}

	if *restoreFromBlob != "" {
		err = wait.PollImmediate(time.Second, 1*time.Hour, func() (bool, error) {
			err = adminClient.Restore(ctx, conf.ResourceGroup, conf.ResourceGroup, *restoreFromBlob)
			if isConnectionRefused(err) {
				return false, nil
			}
			if err != nil {
				return false, err
			}
			return true, nil
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	var oc *v20180930preview.OpenShiftManagedCluster
	err = wait.PollImmediate(time.Second, 1*time.Hour, func() (bool, error) {
		if oc, err = execute(ctx, log, adminClient, v20180930previewClient, conf, *adminManifest); err != nil {
			if isConnectionRefused(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if err != nil {
		log.Fatal(err)
	}

	if !isUpdate {
		if err := updateAadApplication(ctx, oc, log, conf); err != nil {
			log.Fatal(err)
		}
	}

	fmt.Printf("\nCluster available at https://%s/\n", *oc.Properties.PublicHostname)
}
