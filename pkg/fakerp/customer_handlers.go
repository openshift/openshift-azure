package fakerp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"

	internalapi "github.com/openshift/openshift-azure/pkg/api"
	v20190430 "github.com/openshift/openshift-azure/pkg/api/2019-04-30"
	admin "github.com/openshift/openshift-azure/pkg/api/admin"
	"github.com/openshift/openshift-azure/pkg/fakerp/shared"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

func (s *Server) handleDelete(w http.ResponseWriter, req *http.Request) {
	cs := req.Context().Value(ContainerServicesKey).(*internalapi.OpenShiftManagedCluster)

	authorizer, err := azureclient.GetAuthorizerFromContext(req.Context(), internalapi.ContextKeyClientAuthorizer)
	if err != nil {
		s.badRequest(w, fmt.Sprintf("Failed to determine request credentials: %v", err))
		return
	}
	// TODO: Determine subscription ID from the request path
	gc := resources.NewGroupsClient(os.Getenv("AZURE_SUBSCRIPTION_ID"))
	gc.Authorizer = authorizer

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

	future, err := gc.Delete(req.Context(), resourceGroup)
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

	s.writeState(internalapi.Deleting)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	if err := future.WaitForCompletionRef(ctx, gc.Client); err != nil {
		s.badRequest(w, fmt.Sprintf("Failed to wait for resource group deletion: %v", err))
		return
	}
	resp, err := future.Result(gc)
	if err != nil {
		s.badRequest(w, fmt.Sprintf("Failed to get resource group deletion response: %v", err))
		return
	}
	// If the resource group deletion is successful, cleanup the object
	// from the memory so the next GET from the client waiting for this
	// long-running operation can exit successfully.
	if resp.StatusCode == http.StatusOK {
		s.log.Infof("deleted resource group %s", resourceGroup)
		s.store.Delete(ContainerServicesKey)
	}
	// And last but not least, we have accepted this DELETE request
	// and are processing it in the background.
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleGet(w http.ResponseWriter, req *http.Request) {
	s.reply(w, req)
}

func (s *Server) handlePut(w http.ResponseWriter, req *http.Request) {
	oldCs := req.Context().Value(ContainerServicesKey).(*internalapi.OpenShiftManagedCluster)

	var err error
	if !shared.IsUpdate() {
		s.writeState(internalapi.Creating)
	} else {
		s.writeState(internalapi.Updating)
	}

	// TODO: Align with the production RP once it supports the admin API
	isAdminRequest := strings.HasPrefix(req.URL.Path, "/admin")

	// convert the external API manifest into the internal API representation
	s.log.Info("read request and convert to internal")
	var cs *internalapi.OpenShiftManagedCluster
	if isAdminRequest {
		var oc *admin.OpenShiftManagedCluster
		oc, err = s.readAdminRequest(req.Body)
		if err == nil {
			cs, err = admin.ToInternal(oc, oldCs)
		}
	} else {
		var oc *v20190430.OpenShiftManagedCluster
		oc, err = s.read20190430Request(req.Body)
		if err == nil {
			cs, err = v20190430.ToInternal(oc, oldCs)
		}
	}
	if err != nil {
		s.badRequest(w, fmt.Sprintf("Failed to convert to internal type: %v", err))
		return
	}
	s.write(cs)

	// apply the request
	cs, err = createOrUpdateWrapper(req.Context(), s.plugin, s.log, cs, oldCs, isAdminRequest, s.testConfig)
	if err != nil {
		s.writeState(internalapi.Failed)
		s.badRequest(w, fmt.Sprintf("Failed to apply request: %v", err))
		return
	}
	s.write(cs)
	s.writeState(internalapi.Succeeded)
	// TODO: Should return status.Accepted similar to how we handle DELETEs
	s.reply(w, req)
}
