package azureclient

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"
)

// VirtualMachineScaleSetsClient is a minimal interface for azure VirtualMachineScaleSetsClient
type VirtualMachineScaleSetsClient interface {
	VirtualMachineScaleSetsClientAddons
	Client
}

type virtualMachineScaleSetsClient struct {
	compute.VirtualMachineScaleSetsClient
}

var _ VirtualMachineScaleSetsClient = &virtualMachineScaleSetsClient{}

// NewVirtualMachineScaleSetsClient creates a new VirtualMachineScaleSetsClient
func NewVirtualMachineScaleSetsClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) VirtualMachineScaleSetsClient {
	client := compute.NewVirtualMachineScaleSetsClient(subscriptionID)
	setupClient(ctx, log, "compute.VirtualMachineScaleSetsClient", &client.Client, authorizer)
	client.PollingDuration = 30 * time.Minute

	return &virtualMachineScaleSetsClient{
		VirtualMachineScaleSetsClient: client,
	}
}

func (c *virtualMachineScaleSetsClient) Client() autorest.Client {
	return c.VirtualMachineScaleSetsClient.Client
}

// VirtualMachineScaleSetVMsClient is a minimal interface for azure VirtualMachineScaleSetVMsClient
type VirtualMachineScaleSetVMsClient interface {
	VirtualMachineScaleSetVMsClientAddons
}

type virtualMachineScaleSetVMsClient struct {
	compute.VirtualMachineScaleSetVMsClient
}

var _ VirtualMachineScaleSetVMsClient = &virtualMachineScaleSetVMsClient{}

// NewVirtualMachineScaleSetVMsClient creates a new VirtualMachineScaleSetVMsClient
func NewVirtualMachineScaleSetVMsClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) VirtualMachineScaleSetVMsClient {
	client := compute.NewVirtualMachineScaleSetVMsClient(subscriptionID)
	setupClient(ctx, log, "compute.VirtualMachineScaleSetVMsClient", &client.Client, authorizer)
	client.PollingDuration = 30 * time.Minute

	return &virtualMachineScaleSetVMsClient{
		VirtualMachineScaleSetVMsClient: client,
	}
}

func (c *virtualMachineScaleSetVMsClient) Client() autorest.Client {
	return c.VirtualMachineScaleSetVMsClient.Client
}

// VirtualMachineScaleSetExtensionsClient is a minimal interface for azure VirtualMachineScaleSetExtensionsClient
type VirtualMachineScaleSetExtensionsClient interface {
	CreateOrUpdate(ctx context.Context, resourceGroupName string, VMScaleSetName string, vmssExtensionName string, extensionParameters compute.VirtualMachineScaleSetExtension) (result compute.VirtualMachineScaleSetExtensionsCreateOrUpdateFuture, err error)
	Get(ctx context.Context, resourceGroupName string, VMScaleSetName string, vmssExtensionName string, expand string) (result compute.VirtualMachineScaleSetExtension, err error)
	List(ctx context.Context, resourceGroupName string, VMScaleSetName string) (result compute.VirtualMachineScaleSetExtensionListResultPage, err error)
	Client
}

type virtualMachineScaleSetExtensionsClient struct {
	compute.VirtualMachineScaleSetExtensionsClient
}

var _ VirtualMachineScaleSetExtensionsClient = &virtualMachineScaleSetExtensionsClient{}

// NewVirtualMachineScaleSetExtensionsClient creates a new VirtualMachineScaleSetExtensionsClient
func NewVirtualMachineScaleSetExtensionsClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) VirtualMachineScaleSetExtensionsClient {
	client := compute.NewVirtualMachineScaleSetExtensionsClient(subscriptionID)
	setupClient(ctx, log, "compute.VirtualMachineScaleSetExtensionsClient", &client.Client, authorizer)

	return &virtualMachineScaleSetExtensionsClient{
		VirtualMachineScaleSetExtensionsClient: client,
	}
}

func (c *virtualMachineScaleSetExtensionsClient) Client() autorest.Client {
	return c.VirtualMachineScaleSetExtensionsClient.Client
}
