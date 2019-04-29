package compute

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

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
	azureclient.SetupClient(ctx, log, "compute.VirtualMachineScaleSetVMsClient", &client.Client, authorizer)
	client.PollingDuration = 30 * time.Minute

	return &virtualMachineScaleSetVMsClient{
		VirtualMachineScaleSetVMsClient: client,
	}
}

func (c *virtualMachineScaleSetVMsClient) Client() autorest.Client {
	return c.VirtualMachineScaleSetVMsClient.Client
}
