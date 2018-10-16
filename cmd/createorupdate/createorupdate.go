package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	"github.com/openshift/openshift-azure/pkg/fakerp"
)

type server struct {
	// the server will not process more than a single
	// request at all times.
	inProgress chan struct{}
	log        *logrus.Entry
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
	httpServer := &http.Server{Addr: "localhost:8080"}
	s.log.Info("starting server on localhost:8080")
	s.log.WithError(httpServer.ListenAndServe()).Warn("Server exited.")
}

// ServeHTTP handles an incoming request to the server.
func (s *server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	ok := s.validate(w, req)
	if !ok {
		return
	}
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

	oc, err = fakerp.CreateOrUpdate(ctx, oc, s.log, config)
	if err != nil {
		resp := "400 Bad Request: Failed to apply request"
		s.log.Debug(resp, err)
		http.Error(w, resp, http.StatusBadRequest)
		return
	}

	res, err := yaml.Marshal(oc)
	if err != nil {
		resp := "500 Internal Server Error: Failed to marshal response"
		s.log.Debug(resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}
	w.Write(res)
}

func (s *server) validate(w http.ResponseWriter, r *http.Request) bool {
	// TODO: Support DELETE and GET
	if r.Method != http.MethodPut {
		resp := "405 Method not allowed"
		s.log.Debug(resp)
		http.Error(w, resp, http.StatusMethodNotAllowed)
		return false
	}

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

	return true
}

func main() {
	// simulate the RP
	s := newServer(os.Getenv("RESOURCEGROUP"))
	go s.ListenAndServe()

	// simulate the API call to the RP
	in, err := ioutil.ReadFile("_data/manifest.yaml")
	if err != nil {
		logrus.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPut, "http://localhost:8080", bytes.NewReader(in))
	if err != nil {
		logrus.Fatal(err)
	}
	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		logrus.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logrus.Fatalf("unexpected status: %s", resp.Status)
	}

	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Fatal(err)
	}
	if err := ioutil.WriteFile("_data/manifest.yaml", out, 0666); err != nil {
		logrus.Fatal(err)
	}
}
