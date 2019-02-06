package fakerp

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
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
