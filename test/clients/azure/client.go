package azure

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/onsi/ginkgo/config"

	"github.com/openshift/openshift-azure/pkg/fakerp/shared"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	externalapi "github.com/openshift/openshift-azure/pkg/util/azureclient/openshiftmanagedcluster/2018-09-30-preview"
	adminapi "github.com/openshift/openshift-azure/pkg/util/azureclient/openshiftmanagedcluster/admin"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
	"github.com/openshift/openshift-azure/pkg/util/configblob"
)

// rpFocus represents the supported RP APIs which e2e tests use to create their azure clients,
// The client will be configured to work either against the real, fake or admin apis
type rpFocus string

var (
	adminRpFocus = rpFocus(regexp.QuoteMeta("[Admin]"))
	fakeRpFocus  = rpFocus(regexp.QuoteMeta("[Fake]"))
	realRpFocus  = rpFocus(regexp.QuoteMeta("[Real]"))
)

func (tf rpFocus) match(focusString string) bool {
	return strings.Contains(focusString, string(tf))
}

// Client is the main controller for azure client objects
type Client struct {
	Accounts                         azureclient.AccountsClient
	Applications                     azureclient.ApplicationsClient
	BlobStorage                      storage.BlobStorageClient
	OpenShiftManagedClusters         externalapi.OpenShiftManagedClustersClient
	OpenShiftManagedClustersAdmin    adminapi.OpenShiftManagedClustersClient
	VirtualMachineScaleSets          azureclient.VirtualMachineScaleSetsClient
	VirtualMachineScaleSetExtensions azureclient.VirtualMachineScaleSetExtensionsClient
	VirtualMachineScaleSetVMs        azureclient.VirtualMachineScaleSetVMsClient
	Resources                        azureclient.ResourcesClient
	VirtualNetworks                  azureclient.VirtualNetworksClient
	VirtualNetworksPeerings          azureclient.VirtualNetworksPeeringsClient
	Groups                           azureclient.GroupsClient
}

// NewClientFromEnvironment creates a new azure client from environment variables.
// Setting the storage client is optional and should only be used selectively by
// tests that need access to the config storage blob because configblob.GetService
// makes api calls to Azure in order to setup the blob client.
func NewClientFromEnvironment(setStorageClient bool) (*Client, error) {
	authorizer, err := azureclient.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, err
	}

	cfg := &cloudprovider.Config{
		TenantID:        os.Getenv("AZURE_TENANT_ID"),
		SubscriptionID:  os.Getenv("AZURE_SUBSCRIPTION_ID"),
		AadClientID:     os.Getenv("AZURE_CLIENT_ID"),
		AadClientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
		ResourceGroup:   os.Getenv("RESOURCEGROUP"),
		Location:        os.Getenv("AZURE_REGION"),
	}
	subscriptionID := cfg.SubscriptionID

	var storageClient storage.BlobStorageClient
	if setStorageClient {
		storageClient, err = configblob.GetService(context.Background(), cfg)
		if err != nil {
			return nil, err
		}
	}

	var rpURL string
	focus := config.GinkgoConfig.FocusString
	switch {
	case adminRpFocus.match(focus), fakeRpFocus.match(focus):
		fmt.Println("configuring the fake resource provider")
		rpURL = fmt.Sprintf("http://%s", shared.LocalHttpAddr)
	case realRpFocus.match(focus):
		fmt.Println("configuring the real resource provider")
		rpURL = externalapi.DefaultBaseURI
	default:
		panic(fmt.Sprintf("invalid focus %q - need to -ginkgo.focus=\\[Admin\\], -ginkgo.focus=\\[Fake\\] or -ginkgo.focus=\\[Real\\]", config.GinkgoConfig.FocusString))
	}

	rpc := externalapi.NewOpenShiftManagedClustersClientWithBaseURI(rpURL, subscriptionID)
	rpc.Authorizer = authorizer

	rpcAdmin := adminapi.NewOpenShiftManagedClustersClientWithBaseURI(rpURL+shared.AdminContext, subscriptionID)
	rpcAdmin.Authorizer = authorizer

	return &Client{
		Accounts:                         azureclient.NewAccountsClient(subscriptionID, authorizer, nil),
		Applications:                     azureclient.NewApplicationsClient(subscriptionID, authorizer, nil),
		BlobStorage:                      storageClient,
		OpenShiftManagedClusters:         rpc,
		OpenShiftManagedClustersAdmin:    rpcAdmin,
		VirtualMachineScaleSets:          azureclient.NewVirtualMachineScaleSetsClient(subscriptionID, authorizer, nil),
		VirtualMachineScaleSetExtensions: azureclient.NewVirtualMachineScaleSetExtensionsClient(subscriptionID, authorizer, nil),
		VirtualMachineScaleSetVMs:        azureclient.NewVirtualMachineScaleSetVMsClient(subscriptionID, authorizer, nil),
		Resources:                        azureclient.NewResourcesClient(subscriptionID, authorizer, nil),
		VirtualNetworks:                  azureclient.NewVirtualNetworkClient(subscriptionID, authorizer, nil),
		VirtualNetworksPeerings:          azureclient.NewVirtualNetworksPeeringsClient(subscriptionID, authorizer, nil),
		Groups:                           azureclient.NewGroupsClient(subscriptionID, authorizer, nil),
	}, nil
}
