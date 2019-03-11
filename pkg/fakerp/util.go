package fakerp

import (
	"fmt"
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

func (s *Server) isAdminRequest(req *http.Request) bool {
	// TODO: Align with the production RP once it supports the admin API
	return strings.HasPrefix(req.URL.Path, "/admin")
}
