package fakerp

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
)

func (s *Server) badRequest(w http.ResponseWriter, msg string) {
	resp := fmt.Sprintf("400 Bad Request: %s", msg)
	s.log.Debug(resp)
	http.Error(w, resp, http.StatusBadRequest)
}

func (s *Server) internalError(w http.ResponseWriter, msg string) {
	resp := fmt.Sprintf("500 Internal Error: %s", msg)
	s.log.Debug(resp)
	http.Error(w, resp, http.StatusInternalServerError)
}

func readBlobName(req *http.Request) (string, error) {
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read request body: %v", err)
	}
	return strings.Trim(string(data), "\""), nil
}

func readCommandInput(req *http.Request) (*compute.RunCommandInput, error) {
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %v", err)
	}
	var params *compute.RunCommandInput
	err = json.Unmarshal(data, params)
	return params, err
}

func (s *Server) isAdminRequest(req *http.Request) bool {
	// TODO: Align with the production RP once it supports the admin API
	return strings.HasPrefix(req.URL.Path, "/admin")
}
