package fakerp

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/ghodss/yaml"
	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	internalapi "github.com/openshift/openshift-azure/pkg/api"
	v20190430 "github.com/openshift/openshift-azure/pkg/api/2019-04-30"
	v20190930preview "github.com/openshift/openshift-azure/pkg/api/2019-09-30-preview"
	v20191027preview "github.com/openshift/openshift-azure/pkg/api/2019-10-27-preview"
	admin "github.com/openshift/openshift-azure/pkg/api/admin"
	"github.com/openshift/openshift-azure/pkg/fakerp/store"
	"github.com/openshift/openshift-azure/pkg/plugin"
)

const latestApiVersion = "2019-09-30-preview"

type Server struct {
	router *chi.Mux
	// the server will not process more than a single
	// PUT request at all times.
	inProgress chan struct{}

	store store.Store

	log      *logrus.Entry
	address  string
	basePath string

	plugin     internalapi.Plugin
	testConfig api.TestConfig
}

func NewServer(log *logrus.Entry, resourceGroup, address string) *Server {
	s := &Server{
		router:     chi.NewRouter(),
		inProgress: make(chan struct{}, 1),
		log:        log,
		address:    address,
		store:      store.New(log, "_data"),
		basePath:   "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{provider}/openShiftManagedClusters/{resourceName}",
	}

	var errs []error
	var err error
	s.testConfig = GetTestConfig()
	pluginTemplate, err := GetPluginTemplate()
	if err != nil {
		s.log.Fatal(err)
	}
	overridePluginTemplate(pluginTemplate)
	// we dont't know the region/location at this point so we can't load PROXYURL_%region
	// and the plugin keeps the testConfig
	s.plugin, errs = plugin.NewPlugin(s.log, pluginTemplate, s.testConfig)
	if len(errs) > 0 {
		s.log.Fatal(errs)
	}
	errs = s.plugin.ValidatePluginTemplate(context.Background())
	if len(errs) > 0 {
		s.log.Fatal(errs)
	}
	return s
}

func (s *Server) Run() {
	s.setupRoutes()
	s.log.Infof("starting server on %s", s.address)
	s.log.WithError(http.ListenAndServe(s.address, s.router)).Warn("Server exited.")
}

func (s *Server) read20190430Request(body io.ReadCloser, oldCs *api.OpenShiftManagedCluster) (*api.OpenShiftManagedCluster, error) {
	data, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %v", err)
	}
	var oc *v20190430.OpenShiftManagedCluster
	if err := yaml.Unmarshal(data, &oc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %v", err)
	}
	return v20190430.ToInternal(oc, oldCs)
}

func (s *Server) read20190930Request(body io.ReadCloser, oldCs *api.OpenShiftManagedCluster) (*api.OpenShiftManagedCluster, error) {
	data, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %v", err)
	}
	var oc *v20190930preview.OpenShiftManagedCluster
	if err := yaml.Unmarshal(data, &oc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %v", err)
	}
	return v20190930preview.ToInternal(oc, oldCs)
}

func (s *Server) read20191027Request(body io.ReadCloser, oldCs *api.OpenShiftManagedCluster) (*api.OpenShiftManagedCluster, error) {
	data, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %v", err)
	}
	var oc *v20191027preview.OpenShiftManagedCluster
	if err := yaml.Unmarshal(data, &oc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %v", err)
	}
	return v20191027preview.ToInternal(oc, oldCs)
}

func (s *Server) readAdminRequest(body io.ReadCloser, oldCs *api.OpenShiftManagedCluster) (*api.OpenShiftManagedCluster, error) {
	data, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %v", err)
	}
	var oc *admin.OpenShiftManagedCluster
	if err := yaml.Unmarshal(data, &oc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %v", err)
	}
	return admin.ToInternal(oc, oldCs)
}
