package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
	s.log.Infof("deleting resource group %s", resourceGroup)

	future, err := gc.Delete(context.Background(), resourceGroup)
	if err != nil {
		resp := "500 Internal Error: Failed to delete resource group"
		s.log.Debug(resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}
	if err := future.WaitForCompletionRef(context.Background(), gc.Client); err != nil {
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
	ctx := context.Background()
	ctx = context.WithValue(ctx, api.ContextKeyClientID, os.Getenv("AZURE_CLIENT_ID"))
	ctx = context.WithValue(ctx, api.ContextKeyClientSecret, os.Getenv("AZURE_CLIENT_SECRET"))
	ctx = context.WithValue(ctx, api.ContextKeyTenantID, os.Getenv("AZURE_TENANT_ID"))

	config := &api.PluginConfig{
		SyncImage:       os.Getenv("SYNC_IMAGE"),
		LogBridgeImage:  os.Getenv("LOGBRIDGE_IMAGE"),
		AcceptLanguages: []string{"en-us"},
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
	method  = flag.String("request", http.MethodPut, "Specify request to send to the OpenShift resource provider. Supported methods are PUT and DELETE.")
	useProd = flag.Bool("use-prod", false, "If true, send the request to the production OpenShift resource provider.")
)

func validate() error {
	switch strings.ToUpper(*method) {
	case http.MethodPut, http.MethodDelete:
	default:
		return fmt.Errorf("invalid request: %s, Supported methods are PUT and DELETE", strings.ToUpper(*method))
	}
	return nil
}

func main() {
	flag.Parse()
	if err := validate(); err != nil {
		logrus.Fatal(err)
	}

	// simulate the RP
	if !*useProd {
		s := newServer(os.Getenv("RESOURCEGROUP"))
		go s.ListenAndServe()
	}

	// setup the osa client
	rpURL := fmt.Sprintf("http://%s", fakeRpAddr)
	if *useProd {
		rpURL = sdk.DefaultBaseURI
	}
	rpc := sdk.NewOpenShiftManagedClustersClientWithBaseURI(rpURL, os.Getenv("AZURE_SUBSCRIPTION_ID"))

	if strings.ToUpper(*method) == http.MethodDelete {
		future, err := rpc.Delete(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"))
		if err != nil {
			logrus.Fatal(err)
		}
		if err := future.WaitForCompletionRef(context.Background(), rpc.Client); err != nil {
			logrus.Fatal(err)
		}
		resp, err := future.Result(rpc)
		if err != nil {
			logrus.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			logrus.Fatalf("unexpected status: %s, expected 200 OK", resp.Status)
		}
		return
	}

	// simulate the API call to the RP
	in, err := ioutil.ReadFile("_data/manifest.yaml")
	if err != nil {
		logrus.Fatal(err)
	}
	var oc sdk.OpenShiftManagedCluster
	if err := yaml.Unmarshal(in, &oc); err != nil {
		logrus.Fatal(err)
	}
	future, err := rpc.CreateOrUpdate(context.Background(), os.Getenv("RESOURCEGROUP"), os.Getenv("RESOURCEGROUP"), oc)
	if err != nil {
		logrus.Fatal(err)
	}
	if err := future.WaitForCompletionRef(context.Background(), rpc.Client); err != nil {
		logrus.Fatal(err)
	}
	resp, err := future.Result(rpc)
	if err != nil {
		logrus.Fatal(err)
	}
	out, err := yaml.Marshal(resp)
	if err != nil {
		logrus.Fatal(err)
	}
	if err := ioutil.WriteFile("_data/manifest.yaml", out, 0666); err != nil {
		logrus.Fatal(err)
	}
}
