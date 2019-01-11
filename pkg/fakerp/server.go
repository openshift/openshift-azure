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
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/sirupsen/logrus"

	internalapi "github.com/openshift/openshift-azure/pkg/api"
	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	admin "github.com/openshift/openshift-azure/pkg/api/admin/api"
	"github.com/openshift/openshift-azure/pkg/fakerp/shared"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

var once sync.Once

type Server struct {
	router *chi.Mux
	// the server will not process more than a single
	// PUT request at all times.
	inProgress chan struct{}

	gc resources.GroupsClient

	sync.RWMutex
	state internalapi.ProvisioningState
	cs    *internalapi.OpenShiftManagedCluster

	log      *logrus.Entry
	address  string
	basePath string
}

func NewServer(log *logrus.Entry, resourceGroup, address string) *Server {
	s := &Server{
		router:     chi.NewRouter(),
		inProgress: make(chan struct{}, 1),
		log:        log,
		address:    address,
		basePath:   "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{provider}/openShiftManagedClusters/{resourceName}",
	}
	// We need to restore the internal cluster state into memory for GETs
	// and DELETEs to work appropriately.
	if _, err := s.load(); err != nil {
		s.log.Fatal(err)
	}
	return s
}

func (s *Server) Run() {
	s.router.Use(middleware.DefaultCompress)
	s.router.Use(middleware.RedirectSlashes)
	s.router.Use(middleware.Recoverer)
	s.router.Use(s.logger)
	s.router.Use(s.validator)

	s.router.Delete(s.basePath, s.handleDelete)
	s.router.Get(s.basePath, s.handleGet)
	s.router.Put(s.basePath, s.handlePut)
	s.router.Delete(filepath.Join("/admin", s.basePath), s.handleDelete)
	s.router.Get(filepath.Join("/admin", s.basePath), s.handleGet)
	s.router.Put(filepath.Join("/admin", s.basePath), s.handlePut)
	s.router.Put(filepath.Join("/admin", s.basePath, "/restore"), s.handleRestore)
	s.router.Put(filepath.Join("/admin", s.basePath, "/rotate/secrets"), s.handleRotateSecrets)

	s.log.Infof("starting server on %s", s.address)
	s.log.WithError(http.ListenAndServe(s.address, s.router)).Warn("Server exited.")
}

// The way we run the fake RP during development cannot really
// be consistent with how the RP runs in production so we need
// to restore the internal state of the cluster from the
// filesystem. Whether the file that holds the state exists or
// not is returned and any other error that was encountered.
func (s *Server) load() (bool, error) {
	dataDir, err := shared.FindDirectory(shared.DataDirectory)
	if err != nil {
		return false, err
	}
	csFile := filepath.Join(dataDir, "containerservice.yaml")
	if _, err = os.Stat(csFile); err != nil {
		return false, nil
	}
	data, err := ioutil.ReadFile(csFile)
	if err != nil {
		return true, err
	}
	var cs *internalapi.OpenShiftManagedCluster
	if err := yaml.Unmarshal(data, &cs); err != nil {
		return true, err
	}
	s.write(cs)
	return true, nil
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
		s.internalError(w, fmt.Sprintf("Failed to determine request credentials: %v", err))
		return
	}

	// delete dns records
	// TODO: get resource group from request path
	err = DeleteOCPDNS(ctx, os.Getenv("AZURE_SUBSCRIPTION_ID"), os.Getenv("RESOURCEGROUP"), os.Getenv("DNS_RESOURCEGROUP"), os.Getenv("DNS_DOMAIN"))
	if err != nil {
		s.internalError(w, fmt.Sprintf("Failed to delete dns records: %v", err))
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
		s.internalError(w, fmt.Sprintf("Failed to delete resource group: %v", err))
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		if err := future.WaitForCompletionRef(ctx, gc.Client); err != nil {
			s.internalError(w, fmt.Sprintf("Failed to wait for resource group deletion: %v", err))
			return
		}
		resp, err := future.Result(gc)
		if err != nil {
			s.internalError(w, fmt.Sprintf("Failed to get resource group deletion response: %v", err))
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
	headers.Add(autorest.HeaderLocation, fmt.Sprintf("http://%s%s", s.address, req.URL.Path))
	// And last but not least, we have accepted this DELETE request
	// and are processing it in the background.
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleGet(w http.ResponseWriter, req *http.Request) {
	s.reply(w, req)
}

func (s *Server) handlePut(w http.ResponseWriter, req *http.Request) {
	// read old config if it exists
	var oldCs *internalapi.OpenShiftManagedCluster
	var err error
	if !shared.IsUpdate() {
		s.writeState(internalapi.Creating)
	} else {
		s.log.Info("read old config")
		oldCs = s.read()
		if oldCs == nil {
			s.internalError(w, "Failed to read old config: internal state does not exist")
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
		s.badRequest(w, fmt.Sprintf("Failed to convert to internal type: %v", err))
		return
	}
	s.write(cs)

	// populate plugin configuration
	config, err := GetPluginConfig()
	if err != nil {
		s.internalError(w, fmt.Sprintf("Failed to configure plugin: %v", err))
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
		s.badRequest(w, fmt.Sprintf("Failed to apply request: %v", err))
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
		s.internalError(w, fmt.Sprintf("Failed to marshal response: %v", err))
		return
	}
	w.Write(res)
}
