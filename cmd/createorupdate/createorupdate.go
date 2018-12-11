package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	"github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

const (
	// https://developer.microsoft.com/en-us/graph/docs/api-reference/beta/api/application_list
	// To list and patch AAD applications, this code needs to have the clientID
	// of an application with the following permissions:
	// API: Windows Azure Active Directory
	//   Delegated permissions:
	//      Sign in and read user profile
	//      Access the directory as the signed-in user
	clientID = "5935b8e2-3915-409c-bfb2-865b7a9291e0"
)

var (
	method  = flag.String("request", http.MethodPut, "Specify request to send to the OpenShift resource provider. Supported methods are PUT and DELETE.")
	useProd = flag.Bool("use-prod", false, "If true, send the request to the production OpenShift resource provider.")
	timeout = flag.Duration("timeout", 30*time.Minute, "Timeout of the request to the OpenShift resource provider.")
)

func validate() error {
	switch strings.ToUpper(*method) {
	case http.MethodPut, http.MethodDelete:
	default:
		return fmt.Errorf("invalid request: %s, Supported methods are PUT and DELETE", strings.ToUpper(*method))
	}
	return nil
}

func delete(ctx context.Context, log *logrus.Entry, rpc v20180930preview.OpenShiftManagedClustersClient, resourceGroup string, noWait bool) error {
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
		resp, err := future.Result(rpc)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected response: %s", resp.Status)
		}
		log.Info("deleted cluster")
	}
	return nil
}

func createOrUpdate(ctx context.Context, log *logrus.Entry, rpc v20180930preview.OpenShiftManagedClustersClient, resourceGroup string, oc *v20180930preview.OpenShiftManagedCluster, manifestFile string) error {
	log.Info("creating/updating cluster")
	resp, err := rpc.CreateOrUpdateAndWait(ctx, resourceGroup, resourceGroup, *oc)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response: %s", resp.Status)
	}
	log.Info("created/updated cluster")
	return fakerp.WriteClusterConfigToManifest(&resp, manifestFile)
}

func createResourceGroup(conf *fakerp.Config) (bool, error) {
	authorizer, err := azureclient.NewAuthorizer(conf.ClientID, conf.ClientSecret, conf.TenantID)
	if err != nil {
		return false, err
	}
	gc := resources.NewGroupsClient(conf.SubscriptionID)
	gc.Authorizer = authorizer

	if _, err := gc.Get(context.Background(), conf.ResourceGroup); err == nil {
		return false, nil
	}

	var tags map[string]*string
	if !conf.NoGroupTags {
		tags = make(map[string]*string)
		ttl, now := "76h", fmt.Sprintf("%d", time.Now().Unix())
		tags["now"] = &now
		tags["ttl"] = &ttl
		if conf.ResourceGroupTTL != "" {
			if _, err := time.ParseDuration(conf.ResourceGroupTTL); err != nil {
				return false, fmt.Errorf("invalid ttl provided: %q - %v", conf.ResourceGroupTTL, err)
			}
			tags["ttl"] = &conf.ResourceGroupTTL
		}
	}

	rg := resources.Group{Location: &conf.Region, Tags: tags}
	_, err = gc.CreateOrUpdate(context.Background(), conf.ResourceGroup, rg)
	return true, err
}

func execute(ctx context.Context, log *logrus.Entry, rpc v20180930preview.OpenShiftManagedClustersClient, conf *fakerp.Config) error {
	oc, err := fakerp.LoadClusterConfigFromManifest(log, conf.Manifest)
	if err != nil {
		return err
	}
	// simulate the API call to the RP
	dataDir, err := fakerp.FindDirectory(fakerp.DataDirectory)
	if err != nil {
		return err
	}
	defaultManifestFile := filepath.Join(dataDir, "manifest.yaml")
	if err := wait.PollImmediate(time.Second, 1*time.Hour, func() (bool, error) {
		if err := createOrUpdate(ctx, log, rpc, conf.ResourceGroup, oc, defaultManifestFile); err != nil {
			if autoRestErr, ok := err.(autorest.DetailedError); ok {
				if urlErr, ok := autoRestErr.Original.(*url.Error); ok {
					if netErr, ok := urlErr.Err.(*net.OpError); ok {
						if sysErr, ok := netErr.Err.(*os.SyscallError); ok {
							if sysErr.Err == syscall.ECONNREFUSED {
								return false, nil
							}
						}
					}
				}
			}
			return false, err
		}
		return true, nil
	}); err != nil {
		return err
	}

	return nil
}

func updateAadApplication(ctx context.Context, log *logrus.Entry, conf *fakerp.Config) error {
	if len(conf.AADClientID) > 0 && conf.AADClientID != conf.ClientID {
		log.Info("updating the aad application")
		if len(conf.Username) == 0 || len(conf.Password) == 0 {
			log.Fatal("AZURE_USERNAME and AZURE_PASSWORD are required to when updating the aad application")
		}
		authorizer, err := azureclient.NewAuthorizerFromUsernamePassword(conf.Username, conf.Password, clientID, conf.TenantID, azure.PublicCloud.GraphEndpoint)
		if err != nil {
			return err
		}
		aadClient := azureclient.NewRBACApplicationsClient(conf.TenantID, authorizer, []string{"en-us"})
		callbackURL := fmt.Sprintf("https://%s.%s.cloudapp.azure.com/oauth2callback/Azure%%20AD", conf.ResourceGroup, conf.Region)
		conf.AADClientSecret, err = fakerp.UpdateAADAppSecret(ctx, aadClient, conf.AADClientID, callbackURL)
		if err != nil {
			return err
		}
	} else {
		conf.AADClientID = conf.ClientID
		conf.AADClientSecret = conf.ClientSecret
	}
	// set env variable so enrich() still works
	err := os.Setenv("AZURE_AAD_CLIENT_ID", conf.AADClientID)
	if err != nil {
		return err
	}
	err = os.Setenv("AZURE_AAD_CLIENT_SECRET", conf.AADClientSecret)
	if err != nil {
		return err
	}
	return nil
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
	conf, err := fakerp.NewConfig(log, !isDelete)
	if err != nil {
		log.Fatal(err)
	}

	if !isDelete {
		log.Infof("creating resource group %s", conf.ResourceGroup)
		if isCreate, err := createResourceGroup(conf); err != nil {
			log.Fatal(err)
		} else if !isCreate {
			log.Infof("reusing existing resource group %s", conf.ResourceGroup)
		}
	}

	// simulate the RP
	rpURL := v20180930preview.DefaultBaseURI
	if !*useProd {
		rpURL = fakerp.StartServer(log, conf, fakerp.LocalHttpAddr)
	}

	// setup the osa client
	rpc := v20180930preview.NewOpenShiftManagedClustersClientWithBaseURI(rpURL, conf.SubscriptionID)
	authorizer, err := azureclient.NewAuthorizer(conf.ClientID, conf.ClientSecret, conf.TenantID)
	if err != nil {
		log.Fatal(err)
	}
	rpc.Authorizer = authorizer

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if isDelete {
		if err := delete(ctx, log, rpc, conf.ResourceGroup, conf.NoWait); err != nil {
			log.Fatal(err)
		}
		return
	}

	if !fakerp.IsUpdate() {
		if err := updateAadApplication(ctx, log, conf); err != nil {
			log.Fatal(err)
		}
	}

	err = execute(ctx, log, rpc, conf)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Cluster available at https://%s.%s.cloudapp.azure.com/\n", conf.ResourceGroup, conf.Region)
}
