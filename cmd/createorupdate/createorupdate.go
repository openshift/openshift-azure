package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	"github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	sdk "github.com/openshift/openshift-azure/pkg/util/azureclient/osa-go-sdk/services/containerservice/mgmt/2018-09-30-preview/containerservice"
)

const fakeRpAddr = "localhost:8080"

type server struct {
	// the server will not process more than a single
	// PUT request at all times.
	inProgress chan struct{}

	gc resources.GroupsClient

	sync.RWMutex
	state v20180930preview.ProvisioningState
	oc    *v20180930preview.OpenShiftManagedCluster

	log *logrus.Entry
}

func newServer(resourceGroup string) *server {
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	logger.SetLevel(logrus.DebugLevel)

	return &server{
		inProgress: make(chan struct{}, 1),
		log:        logrus.NewEntry(logger).WithFields(logrus.Fields{"resourceGroup": resourceGroup}),
	}
}

func (s *server) ListenAndServe() {
	// TODO: match the request path the real RP would use
	http.Handle("/", s)
	httpServer := &http.Server{Addr: fakeRpAddr}
	s.log.Infof("starting server on %s", fakeRpAddr)
	s.log.WithError(httpServer.ListenAndServe()).Warn("Server exited.")
}

// ServeHTTP handles an incoming request to the server.
func (s *server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	// validate the request
	ok := s.validate(w, req)
	if !ok {
		return
	}

	// process the request
	switch req.Method {
	case http.MethodDelete:
		s.handleDelete(w, req)
	case http.MethodGet:
		s.handleGet(w, req)
	case http.MethodPut:
		s.handlePut(w, req)
	}
}

func (s *server) validate(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPut && r.Method != http.MethodGet && r.Method != http.MethodDelete {
		resp := "405 Method not allowed"
		s.log.Debugf("%s: %s", r.Method, resp)
		http.Error(w, resp, http.StatusMethodNotAllowed)
		return false
	}

	if r.Method == http.MethodPut {
		select {
		case s.inProgress <- struct{}{}:
			// continue
		default:
			// did not get the lock
			resp := "423 Locked: Processing another in-flight request"
			s.log.Debug(resp)
			http.Error(w, resp, http.StatusLocked)
			return false
		}
	}
	return true
}

func (s *server) handleDelete(w http.ResponseWriter, req *http.Request) {
	authorizer, err := azureclient.NewAuthorizer(os.Getenv("AZURE_CLIENT_ID"), os.Getenv("AZURE_CLIENT_SECRET"), os.Getenv("AZURE_TENANT_ID"))
	if err != nil {
		resp := "500 Internal Error: Failed to determine request credentials"
		s.log.Debug(resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}
	subID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	gc := resources.NewGroupsClient(subID)
	gc.Authorizer = authorizer

	resourceGroup := filepath.Base(req.URL.Path)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	s.log.Infof("deleting resource group %s", resourceGroup)

	future, err := gc.Delete(ctx, resourceGroup)
	if err != nil {
		resp := "500 Internal Error: Failed to delete resource group"
		s.log.Debug(resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}
	if err := future.WaitForCompletionRef(ctx, gc.Client); err != nil {
		resp := "500 Internal Error: Failed to wait for resource group deletion"
		s.log.Debug(resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}
	resp, err := future.Result(gc)
	if err != nil {
		resp := "500 Internal Error: Failed to get resource group deletion response"
		s.log.Debug(resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}
	s.log.Infof("deleted resource group %s", resourceGroup)
	w.WriteHeader(resp.StatusCode)
}

func (s *server) handleGet(w http.ResponseWriter, req *http.Request) {
	s.reply(w, req)
}

func (s *server) handlePut(w http.ResponseWriter, req *http.Request) {
	defer func() {
		// drain once we are done processing this request
		<-s.inProgress
	}()

	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		resp := "400 Bad Request: Failed to read request body"
		s.log.Debug(resp, err)
		http.Error(w, resp, http.StatusBadRequest)
		return
	}

	var oc *v20180930preview.OpenShiftManagedCluster
	if err := yaml.Unmarshal(b, &oc); err != nil {
		resp := "400 Bad Request: Failed to unmarshal request"
		s.log.Debug(resp, err)
		http.Error(w, resp, http.StatusBadRequest)
		return
	}
	s.write(oc)

	// simulate Context with property bag
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()
	ctx = context.WithValue(ctx, api.ContextKeyClientID, os.Getenv("AZURE_CLIENT_ID"))
	ctx = context.WithValue(ctx, api.ContextKeyClientSecret, os.Getenv("AZURE_CLIENT_SECRET"))
	ctx = context.WithValue(ctx, api.ContextKeyTenantID, os.Getenv("AZURE_TENANT_ID"))

	tc := api.TestConfig{
		RunningUnderTest:   os.Getenv("RUNNING_UNDER_TEST") != "",
		ImageResourceGroup: os.Getenv("IMAGE_RESOURCEGROUP"),
		ImageResourceName:  os.Getenv("IMAGE_RESOURCENAME"),
		DeployOS:           os.Getenv("DEPLOY_OS"),
		ImageOffer:         os.Getenv("IMAGE_OFFER"),
		ImageVersion:       os.Getenv("IMAGE_VERSION"),
		ORegURL:            os.Getenv("OREG_URL"),
	}

	config := &api.PluginConfig{
		SyncImage:       os.Getenv("SYNC_IMAGE"),
		LogBridgeImage:  os.Getenv("LOGBRIDGE_IMAGE"),
		AcceptLanguages: []string{"en-us"},
		TestConfig:      tc,
	}

	if currentState := s.readState(); string(currentState) == "" {
		s.writeState(v20180930preview.Creating)
	} else {
		// TODO: Need to separate between updates and upgrades
		s.writeState(v20180930preview.Updating)
	}

	if _, err := fakerp.CreateOrUpdate(ctx, oc, s.log, config); err != nil {
		s.writeState(v20180930preview.Failed)
		resp := "400 Bad Request: Failed to apply request"
		s.log.Debug(resp, err)
		http.Error(w, resp, http.StatusBadRequest)
		return
	}
	s.writeState(v20180930preview.Succeeded)
	s.reply(w, req)
}

func (s *server) write(oc *v20180930preview.OpenShiftManagedCluster) {
	s.Lock()
	defer s.Unlock()
	s.oc = oc
}

func (s *server) read() *v20180930preview.OpenShiftManagedCluster {
	s.RLock()
	defer s.RUnlock()
	return s.oc
}

func (s *server) writeState(state v20180930preview.ProvisioningState) {
	s.Lock()
	defer s.Unlock()
	s.state = state
}

func (s *server) readState() v20180930preview.ProvisioningState {
	s.RLock()
	defer s.RUnlock()
	return s.state
}

func (s *server) reply(w http.ResponseWriter, req *http.Request) {
	oc := s.read()
	if oc == nil {
		// This is a delete (trust me)
		// TODO: Need to model this better.
		return
	}
	oc.Properties.ProvisioningState = s.readState()
	res, err := json.Marshal(azureclient.ExternalToSdk(oc))
	if err != nil {
		resp := "500 Internal Server Error: Failed to marshal response"
		s.log.Debug(resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}
	w.Write(res)
}

var (
	method   = flag.String("request", http.MethodPut, "Specify request to send to the OpenShift resource provider. Supported methods are PUT and DELETE.")
	useProd  = flag.Bool("use-prod", false, "If true, send the request to the production OpenShift resource provider.")
	manifest = flag.String("manifest", "_data/manifest.yaml", "Manifest to use for the initial request.")
	update   = flag.String("update", "", "If provided, use this manifest to make a follow-up request after the initial request succeeds.")
	cleanup  = flag.Bool("rm", false, "Delete the cluster once all other requests have completed successfully.")

	// timeouts
	rmTimeout     = flag.Duration("rm-timeout", 20*time.Minute, "Timeout of the cleanup request")
	timeout       = flag.Duration("timeout", 30*time.Minute, "Timeout of the initial request")
	updateTimeout = flag.Duration("update-timeout", 30*time.Minute, "Timeout of the update request")

	// exec hooks
	hook       = flag.String("exec", "", "Command to execute after the initial request to the RP has succeeded.")
	updateHook = flag.String("update-exec", "", "Command to execute after the update request to the RP has succeeded.")

	// TODO: Flag for gathering artifacts from the cluster
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
	return nil
}

func delete(ctx context.Context, log *logrus.Entry, rpc sdk.OpenShiftManagedClustersClient) error {
	log.Info("deleting cluster")
	future, err := rpc.Delete(ctx, os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
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
		return fmt.Errorf("unexpected status: %s, expected 200 OK", resp.Status)
	}
	log.Info("deleted cluster")
	return nil
}

func createOrUpdate(ctx context.Context, log *logrus.Entry, rpc sdk.OpenShiftManagedClustersClient, manifest string) error {
	log.Info("creating/updating cluster")
	in, err := ioutil.ReadFile(manifest)
	if err != nil {
		return err
	}
	var oc sdk.OpenShiftManagedCluster
	if err := yaml.Unmarshal(in, &oc); err != nil {
		return err
	}
	future, err := rpc.CreateOrUpdate(ctx, os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), oc)
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
	out, err := yaml.Marshal(resp)
	if err != nil {
		return err
	}
	log.Info("created/updated cluster")
	return ioutil.WriteFile(manifest, out, 0666)
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

func main() {
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	log := logrus.NewEntry(logger).WithFields(logrus.Fields{"resourceGroup": os.Getenv("RESOURCEGROUP")})

	flag.Parse()
	if err := validate(); err != nil {
		log.Fatal(err)
	}

	// simulate the RP
	if !*useProd {
		log.Info("starting the fake resource provider")
		s := newServer(os.Getenv("RESOURCEGROUP"))
		go s.ListenAndServe()
	}

	// setup the osa client
	rpURL := fmt.Sprintf("http://%s", fakeRpAddr)
	if *useProd {
		rpURL = sdk.DefaultBaseURI
	}
	rpc := sdk.NewOpenShiftManagedClustersClientWithBaseURI(rpURL, os.Getenv("AZURE_SUBSCRIPTION_ID"))
	authorizer, err := azureclient.NewAuthorizer(os.Getenv("AZURE_CLIENT_ID"), os.Getenv("AZURE_CLIENT_SECRET"), os.Getenv("AZURE_TENANT_ID"))
	if err != nil {
		log.Fatal(err)
	}
	rpc.Authorizer = authorizer

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if strings.ToUpper(*method) == http.MethodDelete {
		if err := delete(ctx, log, rpc); err != nil {
			log.Fatal(err)
		}
		return
	}

	// if a cleanup is requested, do it unconditionally at the end
	if *cleanup {
		defer func() {
			delCtx, delCancel := context.WithTimeout(context.Background(), *rmTimeout)
			defer delCancel()
			if err := delete(delCtx, log, rpc); err != nil {
				log.Fatal(err)
			}
		}()
	}

	// simulate the API call to the RP
	if err := createOrUpdate(ctx, log, rpc, *manifest); err != nil {
		log.Fatal(err)
	}

	if *hook != "" {
		if err := execCommand(*hook); err != nil {
			log.Fatal(err)
		}
	}

	// if an update is requested, do it
	if *update != "" {
		updateCtx, updateCancel := context.WithTimeout(context.Background(), *updateTimeout)
		defer updateCancel()
		if err := createOrUpdate(updateCtx, log, rpc, *update); err != nil {
			log.Fatal(err)
		}
	}

	if *updateHook != "" {
		if err := execCommand(*updateHook); err != nil {
			log.Fatal(err)
		}
	}
}
