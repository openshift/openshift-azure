package fakerp

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/go-chi/chi"

	"github.com/openshift/openshift-azure/pkg/cluster"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
	"github.com/openshift/openshift-azure/pkg/util/configblob"
)

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

	cpc := &cloudprovider.Config{
		TenantID:        cs.Properties.AzProfile.TenantID,
		SubscriptionID:  cs.Properties.AzProfile.SubscriptionID,
		AadClientID:     cs.Properties.ServicePrincipalProfile.ClientID,
		AadClientSecret: cs.Properties.ServicePrincipalProfile.Secret,
		ResourceGroup:   cs.Properties.AzProfile.ResourceGroup,
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

	ctx := context.Background()
	bsc, err := configblob.GetService(ctx, cpc)
	if err != nil {
		s.internalError(w, fmt.Sprintf("Failed to configure blob client: %v", err))
		return
	}
	etcdContainer := bsc.GetContainerReference(cluster.EtcdBackupContainerName)

	blob := etcdContainer.GetBlobReference(blobName)
	exists, err := blob.Exists()
	if err != nil {
		s.internalError(w, fmt.Sprintf("Cannot get blob ref for %s: %v", blobName, err))
		return
	}
	if !exists {
		resp, err := etcdContainer.ListBlobs(storage.ListBlobsParameters{})
		if err == nil {
			s.log.Infof("available blobs:")
			for _, blob := range resp.Blobs {
				s.log.Infof("  %s", blob.Name)
			}
		}
		s.badRequest(w, fmt.Sprintf("Blob %s does not exist", blobName))
		return
	}

	ctx, err = enrichContext(context.Background())
	if err != nil {
		s.internalError(w, fmt.Sprintf("Failed to enrich context: %v", err))
		return
	}
	deployer := GetDeployer(s.log, cs, s.pluginConfig)
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
	deployer := GetDeployer(s.log, cs, s.pluginConfig)
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
	deployer := GetDeployer(s.log, cs, s.pluginConfig)
	if err := s.plugin.ForceUpdate(ctx, cs, deployer); err != nil {
		s.internalError(w, fmt.Sprintf("Failed to force update cluster: %v", err))
		return
	}
	s.log.Info("force-updated cluster")
}
