package fakerp

import (
	"fmt"
	"net/http"
	"os"

	internalapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/resources"
)

func (s *Server) handleDelete(w http.ResponseWriter, req *http.Request) {
	cs := req.Context().Value(contextKeyContainerService).(*internalapi.OpenShiftManagedCluster)

	cs.Properties.ProvisioningState = internalapi.Deleting
	s.store.Put(cs)

	s.log.Info("deleting service principals")
	am, err := newAADManager(req.Context(), s.log, cs, s.testConfig)
	if err != nil {
		s.badRequest(w, fmt.Sprintf("Failed to delete service principals: %v", err))
		return
	}

	err = am.deleteApps(req.Context())
	if err != nil {
		s.badRequest(w, fmt.Sprintf("Failed to delete service principals: %v", err))
		return
	}

	s.log.Info("deleting dns records")
	dm, err := newDNSManager(req.Context(), s.log, cs.Properties.AzProfile.SubscriptionID, os.Getenv("DNS_RESOURCEGROUP"), os.Getenv("DNS_DOMAIN"))
	if err != nil {
		s.badRequest(w, fmt.Sprintf("Failed to delete dns records: %v", err))
		return
	}

	err = dm.deleteOCPDNS(req.Context(), cs)
	if err != nil {
		s.badRequest(w, fmt.Sprintf("Failed to delete dns records: %v", err))
		return
	}

	s.log.Infof("deleting resource group")
	authorizer, err := azureclient.GetAuthorizerFromContext(req.Context(), internalapi.ContextKeyClientAuthorizer)
	if err != nil {
		s.badRequest(w, fmt.Sprintf("Failed to determine request credentials: %v", err))
		return
	}

	gc := resources.NewGroupsClient(req.Context(), s.log, cs.Properties.AzProfile.SubscriptionID, authorizer)
	err = gc.Delete(req.Context(), cs.Properties.AzProfile.ResourceGroup)
	if err != nil {
		s.badRequest(w, fmt.Sprintf("Failed to delete resource group: %v", err))
		return
	}

	s.store.Delete()
}

func (s *Server) handleGet(w http.ResponseWriter, req *http.Request) {
	cs := req.Context().Value(contextKeyContainerService).(*internalapi.OpenShiftManagedCluster)
	s.reply(w, req, cs)
}

func (s *Server) handlePut(w http.ResponseWriter, req *http.Request) {
	oldCs := req.Context().Value(contextKeyContainerService).(*internalapi.OpenShiftManagedCluster)

	isAdmin := isAdminRequest(req)

	// convert the external API manifest into the internal API representation
	s.log.Info("read request and convert to internal")
	var cs *internalapi.OpenShiftManagedCluster
	var err error
	if isAdmin {
		cs, err = s.readAdminRequest(req.Body, oldCs)
		if err == nil {
			cs.Properties.ProvisioningState = internalapi.AdminUpdating
			s.store.Put(cs)
		}
	} else {
		cs, err = s.read20190430Request(req.Body, oldCs)
		if err == nil {
			cs.Properties.ProvisioningState = internalapi.Updating
			s.store.Put(cs)
		}
	}
	if err != nil {
		s.badRequest(w, fmt.Sprintf("Failed to convert to internal type: %v", err))
		return
	}

	// apply the request
	newCS, err := createOrUpdateWrapper(req.Context(), s.plugin, s.log, cs, oldCs, isAdmin, s.testConfig)
	if err != nil {
		cs.Properties.ProvisioningState = internalapi.Failed
		s.store.Put(cs)
		s.badRequest(w, fmt.Sprintf("Failed to apply request: %v", err))
		return
	}
	cs = newCS
	cs.Properties.ProvisioningState = internalapi.Succeeded
	s.store.Put(cs)

	s.reply(w, req, cs)
}
