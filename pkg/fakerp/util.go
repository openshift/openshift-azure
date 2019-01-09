package fakerp

import (
	"fmt"
	"net/http"
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
