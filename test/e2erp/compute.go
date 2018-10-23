package e2erp

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

const (
	fakepubkey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7laRyN4B3YZmVrDEZLZoIuUA72pQ0DpGuZBZWykCofIfCPrFZAJgFvonKGgKJl6FGKIunkZL9Us/mV4ZPkZhBlE7uX83AAf5i9Q8FmKpotzmaxN10/1mcnEE7pFvLoSkwqrQSkrrgSm8zaJ3g91giXSbtqvSIj/vk2f05stYmLfhAwNo3Oh27ugCakCoVeuCrZkvHMaJgcYrIGCuFo6q0Pfk9rsZyriIqEa9AtiUOtViInVYdby7y71wcbl0AbbCZsTSqnSoVxm2tRkOsXV6+8X4SnwcmZbao3H+zfO1GBhQOLxJ4NQbzAa8IJh810rYARNLptgmsd4cYXVOSosTX azureuser"
)

// ResourceGroupFromManagedResourceID returns the resource group name from  a managed resource group identifier. The managed resource
// group id for managed applications is formatted as follows `/subscriptions/(.+)/resourceGroups/(.+)`
func ResourceGroupFromManagedResourceID(resourceID string) (string, error) {
	const resourceIDPatternText = `(?i)subscriptions/(.+)/resourceGroups/(.+)`
	resourceIDPattern := regexp.MustCompile(resourceIDPatternText)
	match := resourceIDPattern.FindStringSubmatch(resourceID)
	if len(match) != 3 {
		return "", fmt.Errorf("parsing failed for %s. Invalid managed resource group Id format", resourceID)
	}
	return match[2], nil
}

// ApplicationResourceGroup returns the name of the resource group holding an OpenShift on Azure application
// instance. This resource group may only contain one resource
func ApplicationResourceGroup(resourceGroup, applicationName, location string) string {
	return strings.Join([]string{"OS", resourceGroup, applicationName, location}, "_")
}

// ManagedResourceGroup returns the name of the resource group holding all resources required by an OpenShift on Azure
// managed application instance
func ManagedResourceGroup(ctx context.Context, appsc azureclient.ApplicationsClient, applicationResourceGroup string) (string, error) {
	apps, err := appsc.ListByResourceGroup(ctx, applicationResourceGroup)
	if err != nil {
		return "", err
	}

	for _, app := range apps.Values() {
		if appid := *app.ManagedResourceGroupID; appid != "" {
			r, err := ResourceGroupFromManagedResourceID(appid)
			if err != nil {
				return "", err
			}
			return r, nil
		}
	}
	return "", nil
}

// ScaleSets returns a slice of VirtualMachineScaleSets within a given resource group
func ScaleSets(ctx context.Context, logger *logrus.Entry, ssc azureclient.VirtualMachineScaleSetsClient, resourceGroup string) ([]compute.VirtualMachineScaleSet, error) {
	vmssPages, err := ssc.List(ctx, resourceGroup)
	if err != nil {
		return nil, err
	}

	var scaleSets []compute.VirtualMachineScaleSet
	for vmssPages.NotDone() {
		scaleSets = append(scaleSets, vmssPages.Values()...)
		err = vmssPages.Next()
		if err != nil {
			return nil, err
		}
	}
	return scaleSets, nil
}

// ScaleSetVMs returns a slice of VirtualMachineScaleSetVMs within a given scale set
func ScaleSetVMs(ctx context.Context, logger *logrus.Entry, ssvmc azureclient.VirtualMachineScaleSetVMsClient, resourceGroup string, scaleSet string) ([]compute.VirtualMachineScaleSetVM, error) {
	vmPages, err := ssvmc.List(ctx, resourceGroup, scaleSet, "", "", "")
	if err != nil {
		return nil, err
	}

	var vms []compute.VirtualMachineScaleSetVM
	for vmPages.NotDone() {
		vms = append(vms, vmPages.Values()...)
		err = vmPages.Next()
		if err != nil {
			return nil, err
		}
	}
	return vms, nil
}

// UpdateScaleSetsCapacity returns a slice of the errors it encounters as it attempts to increment the capacity
// (by one) for all scale sets within a given resource group
func UpdateScaleSetsCapacity(ctx context.Context, logger *logrus.Entry, ssc azureclient.VirtualMachineScaleSetsClient, ssvmc azureclient.VirtualMachineScaleSetVMsClient, resourceGroup string) []error {
	var errs []error
	logger.Debugf("listing scale sets for resource group %s", resourceGroup)
	scaleSets, err := ScaleSets(ctx, logger, ssc, resourceGroup)
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	for _, s := range scaleSets {
		logger.Debugf("listing virtual machines in scale set %s", *s.Name)
		vms, err := ScaleSetVMs(ctx, logger, ssvmc, resourceGroup, *s.Name)
		if err != nil {
			errs = append(errs, err)
			return errs
		}
		vmsCount := len(vms)
		logger.Debugf("resizing scale set %s from %d to %d virtual machines", *s.Name, vmsCount, vmsCount+1)
		// we only care about possible errors therefore we do not need to process the returned future
		_, err = ssc.Update(ctx, resourceGroup, *s.Name, compute.VirtualMachineScaleSetUpdate{
			Sku: &compute.Sku{
				Capacity: to.Int64Ptr(int64(vmsCount) + 1),
			},
		})
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// UpdateScaleSetsInstanceType returns a slice of the errors it encounters as it attempts to change the instance
// types for all scale sets within a given resource group
func UpdateScaleSetsInstanceType(ctx context.Context, logger *logrus.Entry, ssc azureclient.VirtualMachineScaleSetsClient, resourceGroup string) []error {
	var errs []error
	logger.Debugf("listing scale sets for resource group %s", resourceGroup)
	scaleSets, err := ScaleSets(ctx, logger, ssc, resourceGroup)
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	for _, s := range scaleSets {
		logger.Debugf("updating instance type for scale set %s from %s to %s", *s.Name, api.StandardD4sV3, api.StandardD2sV3)
		// we only care about possible errors therefore we do not need to process the returned future
		_, err = ssc.Update(ctx, resourceGroup, *s.Name, compute.VirtualMachineScaleSetUpdate{
			Sku: &compute.Sku{
				Name: to.StringPtr(string(api.StandardD2sV3)),
			},
		})
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// UpdateScaleSetSSHKey returns a slice of the errors it encounters as it attempts to update the SSH key for all
// scale sets within a given resource group
func UpdateScaleSetSSHKey(ctx context.Context, logger *logrus.Entry, ssc azureclient.VirtualMachineScaleSetsClient, resourceGroup string) []error {
	var errs []error
	var sshKeyData = fakepubkey
	logger.Debugf("listing scale sets for resource group %s", resourceGroup)
	scaleSets, err := ScaleSets(ctx, logger, ssc, resourceGroup)
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	for _, s := range scaleSets {
		logger.Debugf("updating ssh key for scale set %s", *s.Name)
		// we only care about possible errors therefore we do not need to process the returned future
		_, err := ssc.Update(ctx, resourceGroup, *s.Name, compute.VirtualMachineScaleSetUpdate{
			VirtualMachineScaleSetUpdateProperties: &compute.VirtualMachineScaleSetUpdateProperties{
				VirtualMachineProfile: &compute.VirtualMachineScaleSetUpdateVMProfile{
					OsProfile: &compute.VirtualMachineScaleSetUpdateOSProfile{
						LinuxConfiguration: &compute.LinuxConfiguration{
							SSH: &compute.SSHConfiguration{
								PublicKeys: &[]compute.SSHPublicKey{
									{
										Path:    to.StringPtr("/home/cloud-user/.ssh/authorized_keys"),
										KeyData: to.StringPtr(sshKeyData),
									},
								},
							},
						},
					},
				},
			},
		})
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// RebootScaleSetVMs returns a slice of the errors it encounters as it attempts to reboot all the VMs for all
// scale sets within a given resource group
func RebootScaleSetVMs(ctx context.Context, logger *logrus.Entry, ssc azureclient.VirtualMachineScaleSetsClient, ssvmc azureclient.VirtualMachineScaleSetVMsClient, resourceGroup string) []error {
	var errs []error
	logger.Debugf("listing scale sets in resource group %s", resourceGroup)
	scaleSets, err := ScaleSets(ctx, logger, ssc, resourceGroup)
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	for _, s := range scaleSets {
		logger.Debugf("listing virtual machines in scale set %s", *s.Name)
		vms, err := ScaleSetVMs(ctx, logger, ssvmc, resourceGroup, *s.Name)
		if err != nil {
			errs = append(errs, err)
			return errs
		}
		for _, vm := range vms {
			logger.Debugf("restarting virtual machine %s", *vm.Name)
			// we only care about possible errors therefore we do not need to process the returned future
			_, err := ssvmc.Restart(ctx, resourceGroup, *s.Name, *vm.ID)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs
}

// UpdateScaleSetScriptExtension returns a slice of the errors it encounters as it attempts to set script extensions
// for all scale sets within a given resource group
func UpdateScaleSetScriptExtension(ctx context.Context, logger *logrus.Entry, ssc azureclient.VirtualMachineScaleSetsClient, ssec azureclient.VirtualMachineScaleSetExtensionsClient, resourceGroup string) []error {
	var errs []error
	logger.Debugf("listing scale sets in resource group %s", resourceGroup)
	scaleSets, err := ScaleSets(ctx, logger, ssc, resourceGroup)
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	for _, s := range scaleSets {
		logger.Debugf("updating script extension for scale set %s", *s.Name)
		// we only care about possible errors therefore we do not need to process the returned future
		_, err := ssec.CreateOrUpdate(ctx, resourceGroup, *s.Name, "test", compute.VirtualMachineScaleSetExtension{
			VirtualMachineScaleSetExtensionProperties: &compute.VirtualMachineScaleSetExtensionProperties{
				Type:     to.StringPtr("CustomScript"),
				Settings: `{"fileUris":["https://raw.githubusercontent.com/Azure-Samples/compute-automation-configurations/master/automate_nginx.sh"],"commandToExecute":"./automate_nginx.sh"}`,
			},
		})
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
