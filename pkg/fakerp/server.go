package fakerp

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/ghodss/yaml"
	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	internalapi "github.com/openshift/openshift-azure/pkg/api"
	v20190430 "github.com/openshift/openshift-azure/pkg/api/2019-04-30"
	admin "github.com/openshift/openshift-azure/pkg/api/admin"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin"
	"github.com/openshift/openshift-azure/pkg/fakerp/store"
	"github.com/openshift/openshift-azure/pkg/plugin"
)

var once sync.Once

type Server struct {
	router *chi.Mux
	// the server will not process more than a single
	// PUT request at all times.
	inProgress chan struct{}

	gc resources.GroupsClient

	sync.RWMutex
	store store.Store

	log      *logrus.Entry
	address  string
	basePath string

	plugin         internalapi.Plugin
	testConfig     api.TestConfig
	pluginTemplate *pluginapi.Config
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
	s.pluginTemplate, err = GetPluginTemplate()
	if err != nil {
		s.log.Fatal(err)
	}
	overridePluginTemplate(s.pluginTemplate)
	s.plugin, errs = plugin.NewPlugin(s.log, s.pluginTemplate, s.testConfig)
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

func (s *Server) read20190430Request(body io.ReadCloser) (*v20190430.OpenShiftManagedCluster, error) {
	data, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %v", err)
	}
	var oc *v20190430.OpenShiftManagedCluster
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
