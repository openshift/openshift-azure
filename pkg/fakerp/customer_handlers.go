package fakerp

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	internalapi "github.com/openshift/openshift-azure/pkg/api"
	v20190430 "github.com/openshift/openshift-azure/pkg/api/2019-04-30"
	admin "github.com/openshift/openshift-azure/pkg/api/admin"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/resources"
)

func (s *Server) handleDelete(w http.ResponseWriter, req *http.Request) {
	cs := req.Context().Value(ContainerService).(*internalapi.OpenShiftManagedCluster)

	am, err := newAADManager(req.Context(), s.log, cs, s.testConfig)
	if err != nil {
		s.badRequest(w, fmt.Sprintf("Failed to delete service principals: %v", err))
		return
	}

	s.log.Info("deleting service principals")
	err = am.deleteApps(req.Context())
	if err != nil {
		s.badRequest(w, fmt.Sprintf("Failed to delete service principals: %v", err))
		return
	}

	// delete dns records
	// TODO: get resource group from request path
	dm, err := newDNSManager(req.Context(), s.log, os.Getenv("AZURE_SUBSCRIPTION_ID"), os.Getenv("DNS_RESOURCEGROUP"), os.Getenv("DNS_DOMAIN"))
	if err != nil {
		s.badRequest(w, fmt.Sprintf("Failed to delete dns records: %v", err))
		return
	}
	err = dm.deleteOCPDNS(req.Context(), cs)
	if err != nil {
		s.badRequest(w, fmt.Sprintf("Failed to delete dns records: %v", err))
		return
	}

	resourceGroup := filepath.Base(req.URL.Path)
	s.log.Infof("deleting resource group %s", resourceGroup)

	authorizer, err := azureclient.GetAuthorizerFromContext(req.Context(), internalapi.ContextKeyClientAuthorizer)
	if err != nil {
		s.badRequest(w, fmt.Sprintf("Failed to determine request credentials: %v", err))
		return
	}

	gc := resources.NewGroupsClient(req.Context(), s.log, cs.Properties.AzProfile.SubscriptionID, authorizer)
	err = gc.Delete(req.Context(), resourceGroup)
	if err != nil {
		if autoRestErr, ok := err.(autorest.DetailedError); ok {
			if original, ok := autoRestErr.Original.(*azure.RequestError); ok {
				if original.StatusCode == http.StatusNotFound {
					return
				}
			}
		}
		s.badRequest(w, fmt.Sprintf("Failed to delete resource group: %v", err))
		return
	}

	cs.Properties.ProvisioningState = internalapi.Deleting
	s.store.Put(ContainerServiceKey, cs)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleGet(w http.ResponseWriter, req *http.Request) {
	s.reply(w, req)
}

func (s *Server) handlePut(w http.ResponseWriter, req *http.Request) {
	oldCs := req.Context().Value(ContainerService).(*internalapi.OpenShiftManagedCluster)

	// TODO: Align with the production RP once it supports the admin API
	isAdminRequest := strings.HasPrefix(req.URL.Path, "/admin")

	// convert the external API manifest into the internal API representation
	s.log.Info("read request and convert to internal")
	var cs *internalapi.OpenShiftManagedCluster
	var err error
	if isAdminRequest {
		var oc *admin.OpenShiftManagedCluster
		oc, err := s.readAdminRequest(req.Body)
		if err == nil {
			cs, err = admin.ToInternal(oc, oldCs)
		}
	} else {
		var oc *v20190430.OpenShiftManagedCluster
		oc, err := s.read20190430Request(req.Body)
		if err == nil {
			cs, err = v20190430.ToInternal(oc, oldCs)
		}
	}
	if err != nil {
		s.badRequest(w, fmt.Sprintf("Failed to convert to internal type: %v", err))
		return
	}
	// HACK: We persist new ContainerService early.
	// This will overwrite old copy cs with new req version
	s.store.Put(ContainerServiceKey, cs)

	// apply the request
	cs, err = createOrUpdateWrapper(req.Context(), s.plugin, s.log, cs, oldCs, isAdminRequest, s.testConfig)
	if err != nil {
		oldCs.Properties.ProvisioningState = internalapi.Failed
		s.store.Put(ContainerServiceKey, oldCs)
		s.badRequest(w, fmt.Sprintf("Failed to apply request: %v", err))
		return
	}
	cs.Properties.ProvisioningState = internalapi.Succeeded
	s.store.Put(ContainerServiceKey, cs)
	// TODO: Should return status.Accepted similar to how we handle DELETEs
	s.reply(w, req)
}
