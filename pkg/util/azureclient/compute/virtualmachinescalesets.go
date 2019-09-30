package compute

//go:generate mockgen -destination=../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go github.com/openshift/openshift-azure/pkg/util/azureclient/$GOPACKAGE VirtualMachineScaleSetExtensionsClient,VirtualMachineScaleSetsClient,VirtualMachineScaleSetVMsClient
//go:generate gofmt -s -l -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../../util/mocks/mock_azureclient/mock_$GOPACKAGE/$GOPACKAGE.go

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// VirtualMachineScaleSetsClient is a minimal interface for azure VirtualMachineScaleSetsClient
type VirtualMachineScaleSetsClient interface {
	Get(ctx context.Context, resourceGroupName string, VMScaleSetName string) (result compute.VirtualMachineScaleSet, err error)
	GetInstanceView(ctx context.Context, resourceGroupName string, VMScaleSetName string) (result compute.VirtualMachineScaleSetInstanceView, err error)
	VirtualMachineScaleSetsClientAddons
}

type virtualMachineScaleSetsClient struct {
	compute.VirtualMachineScaleSetsClient
}

var _ VirtualMachineScaleSetsClient = &virtualMachineScaleSetsClient{}

// NewVirtualMachineScaleSetsClient creates a new VirtualMachineScaleSetsClient
func NewVirtualMachineScaleSetsClient(ctx context.Context, log *logrus.Entry, subscriptionID string, authorizer autorest.Authorizer) VirtualMachineScaleSetsClient {
	client := compute.NewVirtualMachineScaleSetsClient(subscriptionID)
	azureclient.SetupClient(ctx, log, "compute.VirtualMachineScaleSetsClient", &client.Client, authorizer)

	return &virtualMachineScaleSetsClient{
		VirtualMachineScaleSetsClient: client,
	}
}

func (c *virtualMachineScaleSetsClient) Client() autorest.Client {
	return c.VirtualMachineScaleSetsClient.Client
}
