package fakerp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"

	"github.com/openshift/openshift-azure/pkg/api"
)

// handleBackup handles admin requests to backup an etcd cluster
func (s *Server) handleBackup(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		s.internalError(w, "Failed to read the internal config")
		return
	}

	backupName := chi.URLParam(req, "backupName")

	ctx, err := enrichContext(context.Background())
	if err != nil {
		s.internalError(w, fmt.Sprintf("Failed to enrich context: %v", err))
		return
	}
	if err := s.plugin.BackupEtcdCluster(ctx, cs, backupName); err != nil {
		s.internalError(w, fmt.Sprintf("Failed to backup cluster: %v", err))
		return
	}
	s.log.Info("backed up cluster")
}

// handleGetControlPlanePods handles admin requests for the list of control plane pods
func (s *Server) handleGetControlPlanePods(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		s.internalError(w, "Failed to read the internal config")
		return
	}
	ctx, err := enrichContext(context.Background())
	if err != nil {
		s.internalError(w, fmt.Sprintf("Failed to enrich context: %v", err))
		return
	}
	pods, err := s.plugin.GetControlPlanePods(ctx, cs)
	if err != nil {
		s.internalError(w, fmt.Sprintf("Failed to fetch control plane pods: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(pods)
	s.log.Info("fetched control plane pods")
}

// handleListClusterVMs handles admin requests for the list of cluster VMs
func (s *Server) handleListClusterVMs(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		s.internalError(w, "Failed to read the internal config")
		return
	}
	ctx, err := enrichContext(context.Background())
	if err != nil {
		s.internalError(w, fmt.Sprintf("Failed to enrich context: %v", err))
		return
	}
	json, err := s.plugin.ListClusterVMs(ctx, cs)
	if err != nil {
		s.internalError(w, fmt.Sprintf("Failed to fetch cluster VMs: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(json)
	s.log.Info("fetched cluster VMs")
}

// handleReimage handles reimaging a vm in the cluster
func (s *Server) handleReimage(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		s.internalError(w, "Failed to read the internal config")
		return
	}

	hostname := chi.URLParam(req, "hostname")

	ctx, err := enrichContext(context.Background())
	if err != nil {
		s.internalError(w, fmt.Sprintf("Failed to enrich context: %v", err))
		return
	}
	if err := s.plugin.Reimage(ctx, cs, hostname); err != nil {
		s.internalError(w, fmt.Sprintf("Failed to reimage vm: %v", err))
		return
	}
	s.log.Infof("reimaged %s", hostname)
}

// handleRestore handles admin requests to restore an etcd cluster from a backup
func (s *Server) handleRestore(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		s.internalError(w, "Failed to read the internal config")
		return
	}

	blobName, err := readBlobName(req)
	if err != nil {
		s.badRequest(w, fmt.Sprintf("Cannot read blob name from request: %v", err))
		return
	}
	if len(blobName) == 0 {
		s.badRequest(w, "Blob name to restore from was not provided")
		return
	}

	ctx, err := enrichContext(context.Background())
	if err != nil {
		s.internalError(w, fmt.Sprintf("Failed to enrich context: %v", err))
		return
	}
	deployer := GetDeployer(s.log, cs)
	if err := s.plugin.RecoverEtcdCluster(ctx, cs, deployer, blobName); err != nil {
		s.internalError(w, fmt.Sprintf("Failed to recover cluster: %v", err))
		return
	}

	s.log.Info("recovered cluster")
}

// handleRotateSecrets handles admin requests for the rotation of cluster secrets
func (s *Server) handleRotateSecrets(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		s.internalError(w, "Failed to read the internal config")
		return
	}
	ctx, err := enrichContext(context.Background())
	if err != nil {
		s.internalError(w, fmt.Sprintf("Failed to enrich context: %v", err))
		return
	}
	deployer := GetDeployer(s.log, cs)
	if err := s.plugin.RotateClusterSecrets(ctx, cs, deployer, s.pluginTemplate); err != nil {
		s.internalError(w, fmt.Sprintf("Failed to rotate cluster secrets: %v", err))
		return
	}
	err = writeHelpers(cs)
	if err != nil {
		s.log.Warnf("could not write helpers: %v", err)
	}
	s.log.Info("rotated cluster secrets")
}

// handleForceUpdate handles admin requests for the force updates of clusters
func (s *Server) handleForceUpdate(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		s.internalError(w, "Failed to read the internal config")
		return
	}
	ctx, err := enrichContext(context.Background())
	if err != nil {
		s.internalError(w, fmt.Sprintf("Failed to enrich context: %v", err))
		return
	}
	deployer := GetDeployer(s.log, cs)
	if err := s.plugin.ForceUpdate(ctx, cs, deployer); err != nil {
		s.internalError(w, fmt.Sprintf("Failed to force update cluster: %v", err))
		return
	}
	s.log.Info("force-updated cluster")
}

// handleRunCommand handles running generic commands on a given vm within a scaleset in the cluster
func (s *Server) handleRunCommand(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		s.internalError(w, "Failed to read the internal config")
		return
	}

	hostname := chi.URLParam(req, "hostname")
	command := chi.URLParam(req, "command")

	ctx, err := enrichContext(context.Background())
	if err != nil {
		s.internalError(w, fmt.Sprintf("Failed to enrich context: %v", err))
		return
	}

	s.log.Infof("running command %s on %s", command, hostname)

	err = s.plugin.RunCommand(ctx, cs, hostname, api.Command(command))
	if err != nil {
		s.internalError(w, fmt.Sprintf("Failed to run command: %v", err))
		return
	}

	s.log.Infof("ran command %s on %s", command, hostname)
}
