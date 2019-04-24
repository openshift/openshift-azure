package fakerp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	v20190430 "github.com/openshift/openshift-azure/pkg/api/2019-04-30"
	admin "github.com/openshift/openshift-azure/pkg/api/admin"
)

func (s *Server) badRequest(w http.ResponseWriter, msg string) {
	resp := fmt.Sprintf("400 Bad Request: %s", msg)
	s.log.Debug(resp)
	http.Error(w, resp, http.StatusBadRequest)
}

func (s *Server) isAdminRequest(req *http.Request) bool {
	// TODO: Align with the production RP once it supports the admin API
	return strings.HasPrefix(req.URL.Path, "/admin")
}

// adminreply returns admin requests data
func (s *Server) adminreply(w http.ResponseWriter, err error, out interface{}) {
	if err != nil {
		s.badRequest(w, err.Error())
		return
	}

	if out == nil {
		return
	}

	if b, ok := out.([]byte); ok {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(b)
		return
	}

	b, err := json.Marshal(out)
	if err != nil {
		s.badRequest(w, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
	return
}

// reply return either admin or external api response
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
		oc := admin.FromInternal(cs)
		res, err = json.Marshal(oc)
	} else {
		oc := v20190430.FromInternal(cs)
		res, err = json.Marshal(oc)
	}
	if err != nil {
		s.badRequest(w, fmt.Sprintf("Failed to marshal response: %v", err))
		return
	}
	w.Write(res)
}
