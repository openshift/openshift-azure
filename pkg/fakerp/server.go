package fakerp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
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

func NewServer(log *logrus.Entry, resourceGroup, address string, c *Config) *Server {
	return &Server{
		inProgress: make(chan struct{}, 1),
		log:        log,
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
		// The way we run the fake RP during development cannot really
		// be consistent with how the RP runs in production so we need
		// to restore the previous request and its provisioning state
		// from the filesystem.
		if ok := s.restore(w, "_data/manifest.yaml"); !ok {
			return
		}
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

func (s *Server) restore(w http.ResponseWriter, path string) bool {
	if _, err := os.Stat(path); err != nil {
		return true
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		resp := "500 Internal Error: Failed to restore OpenShift resource"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return false
	}
	oc := s.readRequest(w, ioutil.NopCloser(bytes.NewReader(data)))
	if oc == nil {
		return false
	}
	s.write(oc)
	return true
}

func (s *Server) readRequest(w http.ResponseWriter, body io.ReadCloser) *v20180930preview.OpenShiftManagedCluster {
	b, err := ioutil.ReadAll(body)
	if err != nil {
		resp := "400 Bad Request: Failed to read request body"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusBadRequest)
		return nil
	}
	var oc *v20180930preview.OpenShiftManagedCluster
	if err := yaml.Unmarshal(b, &oc); err != nil {
		resp := "400 Bad Request: Failed to unmarshal request"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusBadRequest)
		return nil
	}
	return oc
}

func (s *Server) handleDelete(w http.ResponseWriter, req *http.Request) {
	config := &api.PluginConfig{
		AcceptLanguages: []string{"en-us"},
	}

	// simulate Context with property bag
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
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
		resp := "500 Internal Error: Failed to delete dns records"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}

	// TODO: Determine subscription ID from the request path
	gc := resources.NewGroupsClient(s.conf.SubscriptionID)
	gc.Authorizer = authorizer

	resourceGroup := filepath.Base(req.URL.Path)
	s.log.Infof("deleting resource group %s", resourceGroup)

	future, err := gc.Delete(ctx, resourceGroup)
	if err != nil {
		if autoRestErr, ok := err.(autorest.DetailedError); ok {
			if original, ok := autoRestErr.Original.(*azure.RequestError); ok {
				if original.StatusCode == http.StatusNotFound {
					return
				}
			}
		}
		resp := "500 Internal Error: Failed to delete resource group"
		s.log.Debugf("%s: %#v", resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		if err := future.WaitForCompletionRef(ctx, gc.Client); err != nil {
			resp := "500 Internal Error: Failed to wait for resource group deletion"
			s.log.Debugf("%s: %v", resp, err)
			return
		}
		resp, err := future.Result(gc)
		if err != nil {
			resp := "500 Internal Error: Failed to get resource group deletion response"
			s.log.Debugf("%s: %v", resp, err)
			return
		}
		// If the resource group deletion is successful, cleanup the object
		// from the memory so the next GET from the client waiting for this
		// long-running operation can exit successfully.
		if resp.StatusCode == http.StatusOK {
			s.log.Infof("deleted resource group %s", resourceGroup)
			s.write(nil)
		}
	}()
	s.writeState(v20180930preview.Deleting)
	// Update headers with Location so subsequent GET requests know the
	// location to query.
	headers := w.Header()
	headers.Add(autorest.HeaderLocation, fmt.Sprintf("http://%s", s.address))
	// And last but not least, we have accepted this DELETE request
	// and are processing it in the background.
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleGet(w http.ResponseWriter, req *http.Request) {
	s.reply(w, req)
}

func (s *Server) handlePut(w http.ResponseWriter, req *http.Request) {
	defer func() {
		// drain once we are done processing this request
		<-s.inProgress
	}()

	oc := s.readRequest(w, req.Body)
	if oc == nil {
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

	// populate plugin configuration
	config, err := getPluginConfig()
	if err != nil {
		resp := "400 Bad Request: Failed to configure plugin"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusBadRequest)
		return
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
		// If the object is not found in memory then
		// it must have been deleted. Exit successfully.
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
