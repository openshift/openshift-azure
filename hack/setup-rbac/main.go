package main

import (
	"context"
	"os"

	azauthorization "github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/authorization"
)

// ensureRoleDefinitions ensures that the OSA Master and OSA Worker roles are
// correctly defined in a subscription
func ensureRoleDefinitions(ctx context.Context, log *logrus.Entry) error {
	authorizer, err := azureclient.NewAuthorizerFromEnvironment("")
	if err != nil {
		return err
	}

	cli := authorization.NewRoleDefinitionsClient(ctx, log, os.Getenv("AZURE_SUBSCRIPTION_ID"), authorizer)

	_, err = cli.CreateOrUpdate(ctx, "/subscriptions/"+os.Getenv("AZURE_SUBSCRIPTION_ID"), fakerp.OSAMasterRoleDefinitionID, azauthorization.RoleDefinition{
		Name: to.StringPtr(fakerp.OSAMasterRoleDefinitionID),
		RoleDefinitionProperties: &azauthorization.RoleDefinitionProperties{
			RoleName: to.StringPtr("OSA Master"),
			Permissions: &[]azauthorization.Permission{
				{
					Actions: &[]string{
						"Microsoft.Compute/disks/read",
						"Microsoft.Compute/disks/write",
						"Microsoft.Compute/disks/delete",
						"Microsoft.Compute/virtualMachineScaleSets/read",
						"Microsoft.Compute/virtualMachineScaleSets/write",
						"Microsoft.Compute/virtualMachineScaleSets/manualUpgrade/action",
						"Microsoft.Compute/virtualMachineScaleSets/virtualMachines/read",
						"Microsoft.Compute/virtualMachineScaleSets/virtualMachines/write",
						"Microsoft.Compute/virtualMachineScaleSets/virtualMachines/networkInterfaces/read",
						"Microsoft.KeyVault/vaults/read",
						"Microsoft.Network/loadBalancers/read",
						"Microsoft.Network/loadBalancers/write",
						"Microsoft.Network/loadBalancers/delete",
						"Microsoft.Network/loadBalancers/backendAddressPools/join/action",
						"Microsoft.Network/networkSecurityGroups/read",
						"Microsoft.Network/networkSecurityGroups/write",
						"Microsoft.Network/publicIPAddresses/read",
						"Microsoft.Network/publicIPAddresses/write",
						"Microsoft.Network/publicIPAddresses/delete",
						"Microsoft.Network/publicIPAddresses/join/action",
						"Microsoft.Network/virtualNetworks/subnets/read",
						"Microsoft.Network/virtualNetworks/subnets/join/action",
						"Microsoft.Storage/storageAccounts/read", // legacy: BlobDiskController?
						"Microsoft.Storage/storageAccounts/listKeys/action",
					},
				},
			},
			AssignableScopes: &[]string{
				"/subscriptions/" + os.Getenv("AZURE_SUBSCRIPTION_ID"),
			},
		},
	})

	_, err = cli.CreateOrUpdate(ctx, "/subscriptions/"+os.Getenv("AZURE_SUBSCRIPTION_ID"), fakerp.OSAWorkerRoleDefinitionID, azauthorization.RoleDefinition{
		Name: to.StringPtr(fakerp.OSAMasterRoleDefinitionID),
		RoleDefinitionProperties: &azauthorization.RoleDefinitionProperties{
			RoleName: to.StringPtr("OSA Worker"),
			Permissions: &[]azauthorization.Permission{
				{
					Actions: &[]string{
						// Think twice before adding anything to this list:
						// could it be used to subvert the cluster?
						"Microsoft.Compute/virtualMachineScaleSets/read",
						"Microsoft.Compute/virtualMachineScaleSets/virtualMachines/read",
						"Microsoft.Compute/virtualMachineScaleSets/virtualMachines/networkInterfaces/read",
						"Microsoft.Storage/storageAccounts/read", // legacy: BlobDiskController?
					},
				},
			},
			AssignableScopes: &[]string{
				"/subscriptions/" + os.Getenv("AZURE_SUBSCRIPTION_ID"),
			},
		},
	})

	return err
}

func main() {
	if err := ensureRoleDefinitions(context.Background(), logrus.NewEntry(logrus.StandardLogger())); err != nil {
		panic(err)
	}
}
