package fakerp

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/storage"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster"
	"github.com/openshift/openshift-azure/pkg/plugin"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
	"github.com/openshift/openshift-azure/pkg/util/configblob"
)

// handleRestore handles admin requests for restoring etcd.
func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	// validate the request
	ok := s.validateRestore(w, r)
	if !ok {
		return
	}

	s.restore(w, r)
}

func (s *Server) validateRestore(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPut {
		resp := fmt.Sprintf("405 Method not allowed: %s", r.Method)
		s.log.Debug(resp)
		http.Error(w, resp, http.StatusMethodNotAllowed)
		return false
	}

	select {
	case s.inProgress <- struct{}{}:
		// continue
	default:
		// did not get the lock
		resp := "423 Locked: Processing another in-flight request"
		s.log.Debug(resp)
		http.Error(w, resp, http.StatusLocked)
		return false
	}
	return true
}

func (s *Server) restore(w http.ResponseWriter, req *http.Request) {
	defer func() {
		// drain once we are done processing this request
		<-s.inProgress
	}()

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

	config, err := GetPluginConfig()
	if err != nil {
		s.internalError(w, fmt.Sprintf("Failed to configure plugin: %v", err))
		return
	}
	p, errs := plugin.NewPlugin(s.log, config)
	if len(errs) > 0 {
		s.internalError(w, fmt.Sprintf("Failed to configure plugin: %v", err))
		return
	}

	ctx = context.WithValue(ctx, api.ContextKeyClientID, cs.Properties.ServicePrincipalProfile.ClientID)
	ctx = context.WithValue(ctx, api.ContextKeyClientSecret, cs.Properties.ServicePrincipalProfile.Secret)
	ctx = context.WithValue(ctx, api.ContextKeyTenantID, cs.Properties.AzProfile.TenantID)

	deployer := GetDeployer(s.log, cs, config)
	if err := p.RecoverEtcdCluster(ctx, cs, deployer, blobName); err != nil {
		s.internalError(w, fmt.Sprintf("Failed to recover cluster: %v", err))
		return
	}

	s.log.Info("recovered cluster")
}

func readBlobName(req *http.Request) (string, error) {
	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read request body: %v", err)
	}
	return strings.Trim(string(data), "\""), nil
}
