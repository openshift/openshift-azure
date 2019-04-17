package fakerp

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"

	"github.com/openshift/openshift-azure/pkg/api"
)

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

// handleBackup handles admin requests to backup an etcd cluster
func (s *Server) handleBackup(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		s.badRequest(w, "Failed to read the internal config")
		return
	}

	backupName := chi.URLParam(req, "backupName")

	err := s.plugin.BackupEtcdCluster(req.Context(), cs, backupName)
	s.adminreply(w, err, nil)
}

// handleGetControlPlanePods handles admin requests for the list of control plane pods
func (s *Server) handleGetControlPlanePods(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		s.badRequest(w, "Failed to read the internal config")
		return
	}

	pods, err := s.plugin.GetControlPlanePods(req.Context(), cs)
	s.adminreply(w, err, pods)
}

// handleListClusterVMs handles admin requests for the list of cluster VMs
func (s *Server) handleListClusterVMs(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		s.badRequest(w, "Failed to read the internal config")
		return
	}

	vms, err := s.plugin.ListClusterVMs(req.Context(), cs)
	s.adminreply(w, err, vms)
}

// handleReimage handles reimaging a vm in the cluster
func (s *Server) handleReimage(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		s.badRequest(w, "Failed to read the internal config")
		return
	}

	hostname := chi.URLParam(req, "hostname")

	err := s.plugin.Reimage(req.Context(), cs, hostname)
	s.adminreply(w, err, nil)
}

// handleListBackups handles admin requests to list etcd backups
func (s *Server) handleListBackups(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		s.badRequest(w, "Failed to read the internal config")
		return
	}

	backups, pluginErr := s.plugin.ListEtcdBackups(req.Context(), cs)
	var err error
	if pluginErr != nil {
		// TODO: fix this nastiness: https://golang.org/doc/faq#nil_error
		err = pluginErr
	}
	s.adminreply(w, err, backups)
}

// handleRestore handles admin requests to restore an etcd cluster from a backup
func (s *Server) handleRestore(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		s.badRequest(w, "Failed to read the internal config")
		return
	}

	backupName := chi.URLParam(req, "backupName")

	pluginErr := s.plugin.RecoverEtcdCluster(req.Context(), cs, GetDeployer(s.log, cs, s.testConfig), backupName)
	var err error
	if pluginErr != nil {
		// TODO: fix this nastiness: https://golang.org/doc/faq#nil_error
		err = pluginErr
	}
	s.adminreply(w, err, nil)
}

// handleRotateSecrets handles admin requests for the rotation of cluster secrets
func (s *Server) handleRotateSecrets(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		s.badRequest(w, "Failed to read the internal config")
		return
	}

	deployer := GetDeployer(s.log, cs, s.testConfig)
	pluginErr := s.plugin.RotateClusterSecrets(req.Context(), cs, deployer)
	if pluginErr != nil {
		s.badRequest(w, pluginErr.Error())
		return
	}

	err := writeHelpers(s.log, cs)
	s.adminreply(w, err, nil)
}

// handleForceUpdate handles admin requests for the force updates of clusters
func (s *Server) handleForceUpdate(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		s.badRequest(w, "Failed to read the internal config")
		return
	}

	pluginErr := s.plugin.ForceUpdate(req.Context(), cs, GetDeployer(s.log, cs, s.testConfig))
	var err error
	if pluginErr != nil {
		// TODO: fix this nastiness: https://golang.org/doc/faq#nil_error
		err = pluginErr
	}
	s.adminreply(w, err, nil)
}

// handleRunCommand handles running generic commands on a given vm within a scaleset in the cluster
func (s *Server) handleRunCommand(w http.ResponseWriter, req *http.Request) {
	cs := s.read()
	if cs == nil {
		s.badRequest(w, "Failed to read the internal config")
		return
	}

	hostname := chi.URLParam(req, "hostname")
	command := chi.URLParam(req, "command")

	err := s.plugin.RunCommand(req.Context(), cs, hostname, api.Command(command))
	s.adminreply(w, err, nil)
}

// handleGetPluginVersion handles admin requests to get the RP plugin version for OpenShiftManagedClusters
func (s *Server) handleGetPluginVersion(w http.ResponseWriter, req *http.Request) {
	version := s.plugin.GetPluginVersion(req.Context())
	s.adminreply(w, nil, version)
}
