package fakerp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"

	internalapi "github.com/openshift/openshift-azure/pkg/api"
	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	admin "github.com/openshift/openshift-azure/pkg/api/admin/api"
	"github.com/openshift/openshift-azure/pkg/fakerp/shared"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

var once sync.Once

type Server struct {
	// the server will not process more than a single
	// PUT request at all times.
	inProgress chan struct{}

	gc resources.GroupsClient

	sync.RWMutex
	state internalapi.ProvisioningState
	cs    *internalapi.OpenShiftManagedCluster

	log     *logrus.Entry
	address string
}

func NewServer(log *logrus.Entry, resourceGroup, address string) *Server {
	s := &Server{
		inProgress: make(chan struct{}, 1),
		log:        log,
		address:    address,
	}
	// We need to restore the internal cluster state into memory for GETs
	// and DELETEs to work appropriately.
	if err := s.restore(); err != nil {
		if _, ok := err.(*os.PathError); !ok {
			s.log.Fatal(err)
		}
	}
	return s
}

func (s *Server) ListenAndServe() {
	// TODO: match the request path the real RP would use
	http.Handle("/", s)
	http.Handle("/admin", s)
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
		resp := fmt.Sprintf("405 Method not allowed: %s", r.Method)
		s.log.Debug(resp)
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

// The way we run the fake RP during development cannot really
// be consistent with how the RP runs in production so we need
// to restore the internal state of the cluster from the
// filesystem.
func (s *Server) restore() error {
	dataDir, err := shared.FindDirectory(shared.DataDirectory)
	if err != nil {
		return err
	}
	data, err := ioutil.ReadFile(filepath.Join(dataDir, "containerservice.yaml"))
	if err != nil {
		return err
	}
	var cs *internalapi.OpenShiftManagedCluster
	if err := yaml.Unmarshal(data, &cs); err != nil {
		return err
	}
	s.write(cs)
	return nil
}

func (s *Server) read20180930previewRequest(body io.ReadCloser) (*v20180930preview.OpenShiftManagedCluster, error) {
	data, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %v", err)
	}
	var oc *v20180930preview.OpenShiftManagedCluster
	if err := yaml.Unmarshal(data, &oc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %v", err)
	}
	return oc, nil
}

func (s *Server) readAdminRequest(body io.ReadCloser) (*admin.OpenShiftManagedCluster, error) {
	data, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %v", err)
	}
	var oc *admin.OpenShiftManagedCluster
	if err := yaml.Unmarshal(data, &oc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %v", err)
	}
	return oc, nil
}

func (s *Server) handleDelete(w http.ResponseWriter, req *http.Request) {
	// simulate Context with property bag
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	// TODO: Get the azure credentials from the request headers
	ctx = context.WithValue(ctx, internalapi.ContextKeyClientID, os.Getenv("AZURE_CLIENT_ID"))
	ctx = context.WithValue(ctx, internalapi.ContextKeyClientSecret, os.Getenv("AZURE_CLIENT_SECRET"))
	ctx = context.WithValue(ctx, internalapi.ContextKeyTenantID, os.Getenv("AZURE_TENANT_ID"))

	// TODO: Get the azure credentials from the request headers
	authorizer, err := azureclient.NewAuthorizer(os.Getenv("AZURE_CLIENT_ID"), os.Getenv("AZURE_CLIENT_SECRET"), os.Getenv("AZURE_TENANT_ID"))
	if err != nil {
		resp := fmt.Sprintf("500 Internal Error: Failed to determine request credentials: %v", err)
		s.log.Debug(resp)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}

	// delete dns records
	// TODO: get resource group from request path
	err = DeleteOCPDNS(ctx, os.Getenv("AZURE_SUBSCRIPTION_ID"), os.Getenv("RESOURCEGROUP"), os.Getenv("DNS_RESOURCEGROUP"), os.Getenv("DNS_DOMAIN"))
	if err != nil {
		resp := fmt.Sprintf("500 Internal Error: Failed to delete dns records: %v", err)
		s.log.Debug(resp)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}

	// TODO: Determine subscription ID from the request path
	gc := resources.NewGroupsClient(os.Getenv("AZURE_SUBSCRIPTION_ID"))
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
		resp := fmt.Sprintf("500 Internal Error: Failed to delete resource group: %v", err)
		s.log.Debug(resp)
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
	s.writeState(internalapi.Deleting)
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

	// read old config if it exists
	var oldCs *internalapi.OpenShiftManagedCluster
	var err error
	if !shared.IsUpdate() {
		s.writeState(internalapi.Creating)
	} else {
		s.log.Info("read old config")
		dataDir, err := shared.FindDirectory(shared.DataDirectory)
		if err != nil {
			resp := fmt.Sprintf("500 Internal Error: Failed to read old config: %v", err)
			s.log.Debug(resp)
			http.Error(w, resp, http.StatusInternalServerError)
			return
		}
		oldCs, err = managedcluster.ReadConfig(filepath.Join(dataDir, "containerservice.yaml"))
		if err != nil {
			resp := fmt.Sprintf("500 Internal Error: Failed to read old config: %v", err)
			s.log.Debug(resp)
			http.Error(w, resp, http.StatusInternalServerError)
			return
		}
		s.writeState(internalapi.Updating)
	}

	// TODO: Align with the production RP once it supports the admin API
	isAdminRequest := strings.HasPrefix(req.URL.Path, "/admin")

	// convert the external API manifest into the internal API representation
	s.log.Info("read request and convert to internal")
	var cs *internalapi.OpenShiftManagedCluster
	if isAdminRequest {
		var oc *admin.OpenShiftManagedCluster
		oc, err = s.readAdminRequest(req.Body)
		if err == nil {
			cs, err = internalapi.ConvertFromAdmin(oc, oldCs)
		}
	} else {
		var oc *v20180930preview.OpenShiftManagedCluster
		oc, err = s.read20180930previewRequest(req.Body)
		if err == nil {
			cs, err = internalapi.ConvertFromV20180930preview(oc, oldCs)
		}
	}
	if err != nil {
		resp := fmt.Sprintf("400 Bad Request: Failed to convert to internal type: %v", err)
		s.log.Debug(resp)
		http.Error(w, resp, http.StatusBadRequest)
		return
	}
	s.write(cs)

	// populate plugin configuration
	config, err := GetPluginConfig()
	if err != nil {
		resp := fmt.Sprintf("400 Bad Request: Failed to configure plugin: %v", err)
		s.log.Debug(resp)
		http.Error(w, resp, http.StatusBadRequest)
		return
	}

	// simulate Context with property bag
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()
	// TODO: Get the azure credentials from the request headers
	ctx = context.WithValue(ctx, internalapi.ContextKeyClientID, os.Getenv("AZURE_CLIENT_ID"))
	ctx = context.WithValue(ctx, internalapi.ContextKeyClientSecret, os.Getenv("AZURE_CLIENT_SECRET"))
	ctx = context.WithValue(ctx, internalapi.ContextKeyTenantID, os.Getenv("AZURE_TENANT_ID"))

	// apply the request
	cs, err = createOrUpdate(ctx, s.log, cs, oldCs, config, isAdminRequest)
	if err != nil {
		s.writeState(internalapi.Failed)
		resp := fmt.Sprintf("400 Bad Request: Failed to apply request: %v", err)
		s.log.Debug(resp)
		http.Error(w, resp, http.StatusBadRequest)
		return
	}
	s.write(cs)
	s.writeState(internalapi.Succeeded)
	// TODO: Should return status.Accepted similar to how we handle DELETEs
	s.reply(w, req)
}

func (s *Server) write(cs *internalapi.OpenShiftManagedCluster) {
	s.Lock()
	defer s.Unlock()
	s.cs = cs
}

func (s *Server) read() *internalapi.OpenShiftManagedCluster {
	s.RLock()
	defer s.RUnlock()
	return s.cs
}

func (s *Server) writeState(state internalapi.ProvisioningState) {
	s.Lock()
	defer s.Unlock()
	s.state = state
}

func (s *Server) readState() internalapi.ProvisioningState {
	s.RLock()
	defer s.RUnlock()
	return s.state
}

func (s *Server) reply(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		// If the object is not found in memory then
		// it must have been deleted or never existed.
		w.WriteHeader(http.StatusNoContent)
		return
	}
	state := s.readState()
	cs.Properties.ProvisioningState = state

	var res []byte
	var err error
	if strings.HasPrefix(req.URL.Path, "/admin") {
		oc := internalapi.ConvertToAdmin(cs)
		res, err = json.Marshal(oc)
	} else {
		oc := internalapi.ConvertToV20180930preview(cs)
		res, err = json.Marshal(oc)
	}
	if err != nil {
		resp := fmt.Sprintf("500 Internal Server Error: Failed to marshal response: %v", err)
		s.log.Debug(resp)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}
	w.Write(res)
}
