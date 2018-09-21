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

// ServeHTTP handles an incoming request to the server.
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	ok := s.validate(w, r, nil)
	if !ok {
		return
	}
	defer func() {
		// drain once we are done processing this request
		<-s.inProgress
	}()

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		resp := "500 Internal Server Error: Failed to read request body"
		logrus.Debug(resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}

	var oc *v20180930preview.OpenShiftManagedCluster
	if err := yaml.Unmarshal(b, &oc); err != nil {
		resp := "500 Internal Server Error: Failed to unmarshal request"
		logrus.Debug(resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}

	oc, err = fakerp.CreateOrUpdate(r.Context(), oc, s.log)
	if err != nil {
		resp := "500 Internal Server Error: Failed to apply request"
		logrus.Debug(resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}

	responce, err := yaml.Marshal(oc)
	if err != nil {
		resp := "500 Internal Server Error: Failed to marshal response"
		logrus.Debug(resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}
	w.Write(responce)
}

func (s *server) ListenAndServe() {
	http.Handle("/", s)
	httpServer := &http.Server{Addr: ":8080"}
	s.log.Info("starting server on :8080")
	logrus.WithError(httpServer.ListenAndServe()).Warn("Server exited.")
}

func (s *server) validate(w http.ResponseWriter, r *http.Request, secret []byte) bool {
	// TODO: Support DELETE
	if r.Method != http.MethodPut {
		resp := "405 Method not allowed"
		logrus.Debug(resp)
		http.Error(w, resp, http.StatusMethodNotAllowed)
		return false
	}

	/*
		contentType := r.Header.Get("content-type")
		if contentType != "application/json" {
			resp := "400 Bad Request: Server only accepts content-type: application/json"
			logrus.Debug(resp)
			http.Error(w, resp, http.StatusBadRequest)
			return false
		}
	*/

	if err := validateSecret(r, secret); err != nil {
		resp := "403 Forbidden: Invalid Signature"
		logrus.Debug(resp)
		http.Error(w, resp, http.StatusForbidden)
		return false
	}

	// TODO: Maybe support more than a single request?
	// For now if the lock is held
	select {
	case s.inProgress <- struct{}{}:
		// continue
	default:
		// did not get the lock
		resp := "423 Locked: Processing another in-flight request"
		logrus.Debug(resp)
		http.Error(w, resp, http.StatusLocked)
		return false
	}

	return true
}

// TODO: Figure out what scheme to use in a request header
// and compare the secrets here.
func validateSecret(r *http.Request, secret []byte) error {
	return nil
}

func main() {
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	logger.SetLevel(logrus.DebugLevel)
	log := logrus.NewEntry(logger)

	rp := &server{
		inProgress: make(chan struct{}, 1),
		log:        log.WithFields(logrus.Fields{"resourceGroup": os.Getenv("RESOURCEGROUP")}),
	}
	go rp.ListenAndServe()

	// read in the external API manifest.
	b, err := ioutil.ReadFile("_data/manifest.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// prepare request to the server
	req, err := http.NewRequest("PUT", "http://localhost:8080", bytes.NewReader(b))
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	ctx = context.WithValue(ctx, api.ContextKeyClientID, os.Getenv("AZURE_CLIENT_ID"))
	ctx = context.WithValue(ctx, api.ContextKeyClientSecret, os.Getenv("AZURE_CLIENT_SECRET"))
	ctx = context.WithValue(ctx, api.ContextKeyTenantID, os.Getenv("AZURE_TENANT_ID"))
	req = req.WithContext(ctx)

	// do request and read response to get the updated API manifest
	c := &http.Client{}
	// TODO: Maybe implement retries in case the server is not responding yet
	resp, err := c.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// persist the returned (updated) external API manifest.
		if b, err := ioutil.ReadAll(resp.Body); err != nil {
			err = ioutil.WriteFile("_data/manifest.yaml", b, 0666)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
