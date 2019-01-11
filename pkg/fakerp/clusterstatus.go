package fakerp

import (
	"context"
	"fmt"
	"net/http"
)

// handleGetControlPlanePods handles admin requests for the list of control plane pods
func (s *Server) handleGetControlPlanePods(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		s.internalError(w, "Failed to read the internal config")
		return
	}
	ctx := enrichContext(context.Background())
	pods, err := s.plugin.GetControlPlanePods(ctx, cs)
	if err != nil {
		s.internalError(w, fmt.Sprintf("Failed to fetch control plane pods: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(pods)
	s.log.Info("fetched control plane pods")
}
