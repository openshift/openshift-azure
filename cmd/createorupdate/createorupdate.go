package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	"github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

var (
	method  = flag.String("request", http.MethodPut, "Specify request to send to the OpenShift resource provider. Supported methods are PUT and DELETE.")
	useProd = flag.Bool("use-prod", false, "If true, send the request to the production OpenShift resource provider.")
	update  = flag.String("update", "", "If provided, use this manifest to make a follow-up request after the initial request succeeds.")
	cleanup = flag.Bool("rm", false, "Delete the cluster once all other requests have completed successfully.")

	// timeouts
	rmTimeout     = flag.Duration("rm-timeout", 20*time.Minute, "Timeout of the cleanup request")
	timeout       = flag.Duration("timeout", 30*time.Minute, "Timeout of the initial request")
	updateTimeout = flag.Duration("update-timeout", 30*time.Minute, "Timeout of the update request")

	// exec hooks
	hook       = flag.String("exec", "", "Command to execute after the initial request to the RP has succeeded.")
	updateHook = flag.String("update-exec", "", "Command to execute after the update request to the RP has succeeded.")

	artifactDir        = flag.String("artifact-dir", "", "Directory to place artifacts before a cluster is deleted.")
	artifactKubeconfig = flag.String("artifact-kubeconfig", "", "Path to kubeconfig to use for gathering artifacts.")
)

func validate() error {
	switch strings.ToUpper(*method) {
	case http.MethodPut, http.MethodDelete:
	default:
		return fmt.Errorf("invalid request: %s, Supported methods are PUT and DELETE", strings.ToUpper(*method))
	}
	if *method == http.MethodDelete && *update != "" {
		return errors.New("cannot do an update when a DELETE is the initial request")
	}
	if *method == http.MethodDelete && *cleanup {
		return errors.New("cannot request a DELETE and -rm at the same time - use one of the two")
	}
	if *method == http.MethodDelete && (*hook != "" || *updateHook != "") {
		return errors.New("cannot request a DELETE and run an exec hook at the same time")
	}
	if *updateHook != "" && *update == "" {
		return errors.New("cannot exec an update hook when no update request is defined")
	}
	if (*artifactDir == "" && *artifactKubeconfig != "") || (*artifactDir != "" && *artifactKubeconfig == "") {
		return errors.New("both -artifact-dir and -artifact-kubeconfig need to be specified")
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
	future, err := rpc.CreateOrUpdate(ctx, resourceGroup, resourceGroup, *oc)
	if err != nil {
		return err
	}
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
	log.Info("created/updated cluster")
	return fakerp.WriteClusterConfigToManifest(&resp, manifestFile)
}

func execCommand(c string) error {
	args := strings.Split(c, " ")
	cmd := exec.Command(args[0], args[1:]...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s\n%v: %s", stdout.String(), err, stderr.String())
	}
	fmt.Println(stdout.String())
	return nil
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
	oc, err := fakerp.LoadClusterConfigFromManifest(log, "", conf)
	if err != nil {
		return err
	}
	// simulate the API call to the RP
	defaultManifestFile := path.Join(fakerp.DataDirectory, "manifest.yaml")
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

	if *hook != "" {
		if err := execCommand(*hook); err != nil {
			return err
		}
	}

	// if an update is requested, do it
	if *update != "" {
		updateCtx, updateCancel := context.WithTimeout(context.Background(), *updateTimeout)
		defer updateCancel()

		oc, err := fakerp.LoadClusterConfigFromManifest(log, *update, conf)
		if err != nil {
			return err
		}
		updateManifestFile := path.Join(fakerp.DataDirectory, "update.yaml")
		if err := createOrUpdate(updateCtx, log, rpc, conf.ResourceGroup, oc, updateManifestFile); err != nil {
			return err
		}
	}

	if *updateHook != "" {
		if err := execCommand(*updateHook); err != nil {
			return err
		}
	}

	if *artifactDir != "" {
		if err := fakerp.GatherArtifacts(*artifactDir, *artifactKubeconfig); err != nil {
			log.Warnf("could not gather artifacts: %v", err)
		}
	}

	return nil
}

func updateAadApplication(ctx context.Context, log *logrus.Entry, conf *fakerp.Config) error {
	if len(conf.AADClientID) > 0 && conf.AADClientID != conf.ClientID {
		log.Info("updating the aad application")
		if len(conf.Username) == 0 || len(conf.Password) == 0 {
			log.Fatal("AZURE_USERNAME and AZURE_PASSWORD are required to when updating the aad application")
		}
		authorizer, err := azureclient.NewAadAuthorizer(conf.Username, conf.Password, conf.TenantID)
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
	os.Setenv("AZURE_AAD_CLIENT_ID", conf.AADClientID)
	os.Setenv("AZURE_AAD_CLIENT_SECRET", conf.AADClientSecret)
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

	conf, err := fakerp.NewConfig(log)
	if err != nil {
		log.Fatal(err)
	}

	var isCreate bool
	if strings.ToUpper(*method) != http.MethodDelete {
		log.Infof("creating resource group %s", conf.ResourceGroup)
		if isCreate, err = createResourceGroup(conf); err != nil {
			log.Fatal(err)
		} else if !isCreate {
			log.Infof("reusing existing resource group %s", conf.ResourceGroup)
		}
	}

	// simulate the RP
	fakeRpAddr := "localhost:8080"
	if !*useProd {
		log.Info("starting the fake resource provider")
		s := fakerp.NewServer(log, conf.ResourceGroup, fakeRpAddr, conf)
		go s.ListenAndServe()
	}

	// setup the osa client
	rpURL := fmt.Sprintf("http://%s", fakeRpAddr)
	if *useProd {
		rpURL = v20180930preview.DefaultBaseURI
	}
	rpc := v20180930preview.NewOpenShiftManagedClustersClientWithBaseURI(rpURL, conf.SubscriptionID)
	authorizer, err := azureclient.NewAuthorizer(conf.ClientID, conf.ClientSecret, conf.TenantID)
	if err != nil {
		log.Fatal(err)
	}
	rpc.Authorizer = authorizer

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if strings.ToUpper(*method) == http.MethodDelete {
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
		log.Warn(err)
	}

	if *cleanup {
		delCtx, delCancel := context.WithTimeout(context.Background(), *rmTimeout)
		defer delCancel()
		if err := delete(delCtx, log, rpc, conf.ResourceGroup, conf.NoWait); err != nil {
			log.Warn(err)
		}
	}

	if err != nil {
		os.Exit(1)
	}
}
