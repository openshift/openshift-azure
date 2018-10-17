package fakerp

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

type Server struct {
	// the server will not process more than a single
	// PUT request at all times.
	inProgress chan struct{}

	gc resources.GroupsClient

	sync.RWMutex
	state v20180930preview.ProvisioningState
	oc    *v20180930preview.OpenShiftManagedCluster

	log     *logrus.Entry
	address string
	conf    *Config
}

func NewServer(resourceGroup, address string, c *Config) *Server {
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	logger.SetLevel(logrus.DebugLevel)

	return &Server{
		inProgress: make(chan struct{}, 1),
		log:        logrus.NewEntry(logger).WithFields(logrus.Fields{"resourceGroup": resourceGroup}),
		address:    address,
		conf:       c,
	}
}

func (s *Server) ListenAndServe() {
	// TODO: match the request path the real RP would use
	http.Handle("/", s)
	httpServer := &http.Server{Addr: s.address}
	s.log.Infof("starting server on %s", s.address)
	s.log.WithError(httpServer.ListenAndServe()).Warn("Server exited.")
}

// ServeHTTP handles an incoming request to the server.
func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
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

func (s *Server) validate(w http.ResponseWriter, r *http.Request) bool {
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

func (s *Server) handleDelete(w http.ResponseWriter, req *http.Request) {

	config := &api.PluginConfig{
		AcceptLanguages: []string{"en-us"},
	}

	// simulate Context with property bag
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()
	// TODO: Get the azure credentials from the request headers
	ctx = context.WithValue(ctx, api.ContextKeyClientID, s.conf.ClientID)
	ctx = context.WithValue(ctx, api.ContextKeyClientSecret, s.conf.ClientSecret)
	ctx = context.WithValue(ctx, api.ContextKeyTenantID, s.conf.TenantID)

	// TODO: Get the azure credentials from the request headers
	authorizer, err := azureclient.NewAuthorizer(s.conf.ClientID, s.conf.ClientSecret, s.conf.TenantID)
	if err != nil {
		resp := "500 Internal Error: Failed to determine request credentials"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}

	// delete dns records
	err = DeleteOCPDNS(ctx, s.conf.SubscriptionID, s.conf.ResourceGroup, s.conf.DnsResourceGroup, s.conf.DnsDomain, config)
	if err != nil {
		s.log.Fatal(err)
	}

	// TODO: Determine subscription ID from the request path
	gc := resources.NewGroupsClient(s.conf.SubscriptionID)
	gc.Authorizer = authorizer

	resourceGroup := filepath.Base(req.URL.Path)
	s.log.Infof("deleting resource group %s", resourceGroup)

	future, err := gc.Delete(ctx, resourceGroup)
	if err != nil {
		resp := "500 Internal Error: Failed to delete resource group"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}
	if err := future.WaitForCompletionRef(ctx, gc.Client); err != nil {
		resp := "500 Internal Error: Failed to wait for resource group deletion"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}
	resp, err := future.Result(gc)
	if err != nil {
		resp := "500 Internal Error: Failed to get resource group deletion response"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}
	s.log.Infof("deleted resource group %s", resourceGroup)
	w.WriteHeader(resp.StatusCode)
}

func (s *Server) handleGet(w http.ResponseWriter, req *http.Request) {
	s.reply(w, req)
}

func (s *Server) handlePut(w http.ResponseWriter, req *http.Request) {
	defer func() {
		// drain once we are done processing this request
		<-s.inProgress
	}()

	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		resp := "400 Bad Request: Failed to read request body"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusBadRequest)
		return
	}

	var oc *v20180930preview.OpenShiftManagedCluster
	if err := yaml.Unmarshal(b, &oc); err != nil {
		resp := "400 Bad Request: Failed to unmarshal request"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusBadRequest)
		return
	}
	s.write(oc)

	// simulate Context with property bag
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()
	// TODO: Get the azure credentials from the request headers
	ctx = context.WithValue(ctx, api.ContextKeyClientID, s.conf.ClientID)
	ctx = context.WithValue(ctx, api.ContextKeyClientSecret, s.conf.ClientSecret)
	ctx = context.WithValue(ctx, api.ContextKeyTenantID, s.conf.TenantID)

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

	if _, err := CreateOrUpdate(ctx, oc, s.log, config); err != nil {
		s.writeState(v20180930preview.Failed)
		resp := "400 Bad Request: Failed to apply request"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusBadRequest)
		return
	}
	s.writeState(v20180930preview.Succeeded)
	s.reply(w, req)
}

func (s *Server) write(oc *v20180930preview.OpenShiftManagedCluster) {
	s.Lock()
	defer s.Unlock()
	s.oc = oc
}

func (s *Server) read() *v20180930preview.OpenShiftManagedCluster {
	s.RLock()
	defer s.RUnlock()
	return s.oc
}

func (s *Server) writeState(state v20180930preview.ProvisioningState) {
	s.Lock()
	defer s.Unlock()
	s.state = state
}

func (s *Server) readState() v20180930preview.ProvisioningState {
	s.RLock()
	defer s.RUnlock()
	return s.state
}

func (s *Server) reply(w http.ResponseWriter, req *http.Request) {
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
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}
	w.Write(res)
}
