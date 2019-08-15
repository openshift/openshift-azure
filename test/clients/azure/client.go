package azure

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/fakerp/shared"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/compute"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/insights"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/network"
	externalapi "github.com/openshift/openshift-azure/pkg/util/azureclient/openshiftmanagedcluster/2019-04-30"
	adminapi "github.com/openshift/openshift-azure/pkg/util/azureclient/openshiftmanagedcluster/admin"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/resources"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
	"github.com/openshift/openshift-azure/pkg/util/configblob"
)

var RPClient *Client

// Client is the main controller for azure client objects
type Client struct {
	Accounts                         storage.AccountsClient
	ActivityLogs                     insights.ActivityLogsClient
	BlobStorage                      storage.BlobStorageClient
	OpenShiftManagedClusters         externalapi.OpenShiftManagedClustersClient
	OpenShiftManagedClustersAdmin    *adminapi.Client
	VirtualMachineScaleSets          compute.VirtualMachineScaleSetsClient
	VirtualMachineScaleSetExtensions compute.VirtualMachineScaleSetExtensionsClient
	VirtualMachineScaleSetVMs        compute.VirtualMachineScaleSetVMsClient
	Resources                        resources.ResourcesClient
	VirtualNetworks                  network.VirtualNetworksClient
	VirtualNetworksPeerings          network.VirtualNetworksPeeringsClient
	Groups                           resources.GroupsClient
}

// NewClientFromEnvironment creates a new azure client from environment variables and stores it in the RPClient variable
func NewClientFromEnvironment(ctx context.Context, log *logrus.Entry) error {
	authorizer, err := azureclient.NewAuthorizerFromEnvironment("")
	if err != nil {
		return err
	}

	cfg := &cloudprovider.Config{
		TenantID:        os.Getenv("AZURE_TENANT_ID"),
		SubscriptionID:  os.Getenv("AZURE_SUBSCRIPTION_ID"),
		AadClientID:     os.Getenv("AZURE_CLIENT_ID"),
		AadClientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
		ResourceGroup:   os.Getenv("RESOURCEGROUP"),
	}
	subscriptionID := cfg.SubscriptionID

	var storageClient storage.BlobStorageClient
	storageClient, err = configblob.GetService(ctx, log, cfg)
	if err != nil {
		return err
	}

	var rpURL string
	fmt.Println("configuring the fake resource provider")
	rpURL = fmt.Sprintf("http://%s", shared.LocalHttpAddr)

	rpc := externalapi.NewOpenShiftManagedClustersClientWithBaseURI(rpURL, subscriptionID)
	rpc.Authorizer = authorizer

	rpcAdmin := adminapi.NewClient(rpURL, subscriptionID)

	RPClient = &Client{
		Accounts:                         storage.NewAccountsClient(ctx, log, subscriptionID, authorizer),
		ActivityLogs:                     insights.NewActivityLogsClient(ctx, log, subscriptionID, authorizer),
		BlobStorage:                      storageClient,
		OpenShiftManagedClusters:         rpc,
		OpenShiftManagedClustersAdmin:    rpcAdmin,
		VirtualMachineScaleSets:          compute.NewVirtualMachineScaleSetsClient(ctx, log, subscriptionID, authorizer),
		VirtualMachineScaleSetExtensions: compute.NewVirtualMachineScaleSetExtensionsClient(ctx, log, subscriptionID, authorizer),
		VirtualMachineScaleSetVMs:        compute.NewVirtualMachineScaleSetVMsClient(ctx, log, subscriptionID, authorizer),
		Resources:                        resources.NewResourcesClient(ctx, log, subscriptionID, authorizer),
		VirtualNetworks:                  network.NewVirtualNetworkClient(ctx, log, subscriptionID, authorizer),
		VirtualNetworksPeerings:          network.NewVirtualNetworksPeeringsClient(ctx, log, subscriptionID, authorizer),
		Groups:                           resources.NewGroupsClient(ctx, log, subscriptionID, authorizer),
	}
	return nil
}
