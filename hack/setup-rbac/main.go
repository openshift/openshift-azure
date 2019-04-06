package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/authorization/mgmt/2015-07-01/authorization"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

const (
	// IDs picked out of thin air
	osaMasterRoleDefinitionID = "9bc35064-26cf-4536-8e65-40bd22a41071"
	osaWorkerRoleDefinitionID = "7c1a95fb-9825-4039-b67c-a3644e872c04"
)

// listActions lists all the mutating actions carried out on a resource group in
// the last 6 hours, along with the principal that carried out the action
func listActions(ctx context.Context, resourceGroupName string) error {
	authorizer, err := azureclient.NewAuthorizerFromEnvironment("")
	if err != nil {
		return err
	}

	cli := azureclient.NewActivityLogsClient(ctx, os.Getenv("AZURE_SUBSCRIPTION_ID"), authorizer)

	pages, err := cli.List(ctx,
		fmt.Sprintf("eventTimestamp ge '%s' and resourceGroupName eq '%s'",
			time.Now().Add(-6*time.Hour).Format(time.RFC3339),
			resourceGroupName),
		"")
	if err != nil {
		return err
	}

	m := map[string]map[string]struct{}{}

	for pages.NotDone() {
		for _, v := range pages.Values() {
			if m[*v.Caller] == nil {
				m[*v.Caller] = map[string]struct{}{}
			}
			m[*v.Caller][*v.Authorization.Action] = struct{}{}
		}
		err = pages.Next()
		if err != nil {
			return err
		}
	}

	for caller, m := range m {
		fmt.Printf("*** %s\n", caller)
		for action := range m {
			fmt.Printf("    %s\n", action)
		}
		fmt.Println()
	}

	return nil
}

// ensureRoleDefinitions ensures that the OSA Master and OSA Worker roles are
// correctly defined in a subscription
func ensureRoleDefinitions(ctx context.Context) error {
	authorizer, err := azureclient.NewAuthorizerFromEnvironment("")
	if err != nil {
		return err
	}

	cli := authorization.NewRoleDefinitionsClient(os.Getenv("AZURE_SUBSCRIPTION_ID"))
	cli.Authorizer = authorizer

	_, err = cli.CreateOrUpdate(ctx, "/subscriptions/"+os.Getenv("AZURE_SUBSCRIPTION_ID"), osaMasterRoleDefinitionID, authorization.RoleDefinition{
		Name: to.StringPtr(osaMasterRoleDefinitionID),
		Properties: &authorization.RoleDefinitionProperties{
			RoleName: to.StringPtr("OSA Master"),
			Permissions: &[]authorization.Permission{
				{
					Actions: &[]string{
						"Microsoft.Compute/disks/read",
						"Microsoft.Compute/disks/write",
						"Microsoft.Compute/disks/delete",
						"Microsoft.Compute/images/read", // e2e lb test when running from an image
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

	_, err = cli.CreateOrUpdate(ctx, "/subscriptions/"+os.Getenv("AZURE_SUBSCRIPTION_ID"), osaWorkerRoleDefinitionID, authorization.RoleDefinition{
		Name: to.StringPtr(osaMasterRoleDefinitionID),
		Properties: &authorization.RoleDefinitionProperties{
			RoleName: to.StringPtr("OSA Worker"),
			Permissions: &[]authorization.Permission{
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
	if err := ensureRoleDefinitions(context.Background()); err != nil {
		panic(err)
	}
}
