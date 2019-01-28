package fakerp

import (
	"context"
	"fmt"
	"net/http"
)

// handleRotateSecrets handles admin requests for the rotation of cluster secrets
func (s *Server) handleRotateSecrets(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		s.internalError(w, "Failed to read the internal config")
		return
	}
	ctx := enrichContext(context.Background())
	deployer := GetDeployer(s.log, cs, s.pluginConfig)
	if err := s.plugin.RotateClusterSecrets(ctx, cs, deployer, s.pluginTemplate); err != nil {
		s.internalError(w, fmt.Sprintf("Failed to rotate cluster secrets: %v", err))
		return
	}
	s.log.Info("rotated cluster secrets")
}
