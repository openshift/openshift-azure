package fakerp

import (
	"net/http"

	"github.com/go-chi/chi"

	"github.com/openshift/openshift-azure/pkg/api"
)

// handleBackup handles admin requests to backup an etcd cluster
func (s *Server) handleBackup(w http.ResponseWriter, req *http.Request) {
	cs := req.Context().Value(contextKeyContainerService).(*api.OpenShiftManagedCluster)

	backupName := chi.URLParam(req, "backupName")

	err := s.plugin.BackupEtcdCluster(req.Context(), cs, backupName)
	s.adminreply(w, err, nil)
}

// handleGetControlPlanePods handles admin requests for the list of control plane pods
func (s *Server) handleGetControlPlanePods(w http.ResponseWriter, req *http.Request) {
	cs := req.Context().Value(contextKeyContainerService).(*api.OpenShiftManagedCluster)

	pods, err := s.plugin.GetControlPlanePods(req.Context(), cs)
	s.adminreply(w, err, pods)
}

// handleListClusterVMs handles admin requests for the list of cluster VMs
func (s *Server) handleListClusterVMs(w http.ResponseWriter, req *http.Request) {
	cs := req.Context().Value(contextKeyContainerService).(*api.OpenShiftManagedCluster)

	vms, err := s.plugin.ListClusterVMs(req.Context(), cs)
	s.adminreply(w, err, vms)
}

// handleReimage handles reimaging a vm in the cluster
func (s *Server) handleReimage(w http.ResponseWriter, req *http.Request) {
	cs := req.Context().Value(contextKeyContainerService).(*api.OpenShiftManagedCluster)

	hostname := chi.URLParam(req, "hostname")

	err := s.plugin.Reimage(req.Context(), cs, hostname)
	s.adminreply(w, err, nil)
}

// handleListBackups handles admin requests to list etcd backups
func (s *Server) handleListBackups(w http.ResponseWriter, req *http.Request) {
	cs := req.Context().Value(contextKeyContainerService).(*api.OpenShiftManagedCluster)

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
	cs := req.Context().Value(contextKeyContainerService).(*api.OpenShiftManagedCluster)

	backupName := chi.URLParam(req, "backupName")

	pluginErr := s.plugin.RecoverEtcdCluster(req.Context(), cs, GetDeployer(s.log, cs, nil, s.testConfig), backupName)
	var err error
	if pluginErr != nil {
		// TODO: fix this nastiness: https://golang.org/doc/faq#nil_error
		err = pluginErr
	}
	s.adminreply(w, err, nil)
}

// handleRotateSecrets handles admin requests for the rotation of cluster secrets
func (s *Server) handleRotateSecrets(w http.ResponseWriter, req *http.Request) {
	cs := req.Context().Value(contextKeyContainerService).(*api.OpenShiftManagedCluster)

	deployer := GetDeployer(s.log, cs, nil, s.testConfig)
	pluginErr := s.plugin.RotateClusterSecrets(req.Context(), cs, deployer)
	if pluginErr != nil {
		s.badRequest(w, pluginErr.Error())
		return
	}

	s.store.Put(cs)
	err := writeHelpers(s.log, cs)
	s.adminreply(w, err, nil)
}

// handleRotateCertificates handles admin requests for the rotation of cluster secrets
func (s *Server) handleRotateCertificates(w http.ResponseWriter, req *http.Request) {
	cs := req.Context().Value(contextKeyContainerService).(*api.OpenShiftManagedCluster)

	deployer := GetDeployer(s.log, cs, nil, s.testConfig)
	pluginErr := s.plugin.RotateClusterCertificates(req.Context(), cs, deployer)
	if pluginErr != nil {
		s.badRequest(w, pluginErr.Error())
		return
	}
	s.store.Put(cs)
	err := writeHelpers(s.log, cs)
	s.adminreply(w, err, nil)
}

// handleRotateCertificatesAndSecrets handles admin requests for the rotation of cluster secrets
func (s *Server) handleRotateCertificatesAndSecrets(w http.ResponseWriter, req *http.Request) {
	cs := req.Context().Value(contextKeyContainerService).(*api.OpenShiftManagedCluster)

	deployer := GetDeployer(s.log, cs, nil, s.testConfig)
	pluginErr := s.plugin.RotateClusterCertificatesAndSecrets(req.Context(), cs, deployer)
	if pluginErr != nil {
		s.badRequest(w, pluginErr.Error())
		return
	}

	s.store.Put(cs)
	err := writeHelpers(s.log, cs)
	s.adminreply(w, err, nil)
}

// handleForceUpdate handles admin requests for the force updates of clusters
func (s *Server) handleForceUpdate(w http.ResponseWriter, req *http.Request) {
	cs := req.Context().Value(contextKeyContainerService).(*api.OpenShiftManagedCluster)

	pluginErr := s.plugin.ForceUpdate(req.Context(), cs, GetDeployer(s.log, cs, nil, s.testConfig))
	var err error
	if pluginErr != nil {
		// TODO: fix this nastiness: https://golang.org/doc/faq#nil_error
		err = pluginErr
	}
	s.adminreply(w, err, nil)
}

// handleRunCommand handles running generic commands on a given vm within a scaleset in the cluster
func (s *Server) handleRunCommand(w http.ResponseWriter, req *http.Request) {
	cs := req.Context().Value(contextKeyContainerService).(*api.OpenShiftManagedCluster)

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
