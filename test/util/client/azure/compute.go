//+build e2e

package azure

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/api"
	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	"github.com/openshift/openshift-azure/pkg/fakerp"
)

const (
	fakepubkey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7laRyN4B3YZmVrDEZLZoIuUA72pQ0DpGuZBZWykCofIfCPrFZAJgFvonKGgKJl6FGKIunkZL9Us/mV4ZPkZhBlE7uX83AAf5i9Q8FmKpotzmaxN10/1mcnEE7pFvLoSkwqrQSkrrgSm8zaJ3g91giXSbtqvSIj/vk2f05stYmLfhAwNo3Oh27ugCakCoVeuCrZkvHMaJgcYrIGCuFo6q0Pfk9rsZyriIqEa9AtiUOtViInVYdby7y71wcbl0AbbCZsTSqnSoVxm2tRkOsXV6+8X4SnwcmZbao3H+zfO1GBhQOLxJ4NQbzAa8IJh810rYARNLptgmsd4cYXVOSosTX azureuser"
)

// ResourceGroupFromManagedResourceID returns the resource group name from  a managed resource group identifier. The managed resource
// group id for managed applications is formatted as follows `/subscriptions/(.+)/resourceGroups/(.+)`
func (az *Client) ResourceGroupFromManagedResourceID(resourceID string) (string, error) {
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
func (az *Client) ApplicationResourceGroup() string {
	return strings.Join([]string{"OS", az.resourceGroup, az.resourceGroup, az.location}, "_")
}

// ManagedResourceGroup returns the name of the resource group holding all resources required by an OpenShift on Azure
// managed application instance
func (az *Client) ManagedResourceGroup(applicationResourceGroup string) (string, error) {
	apps, err := az.appsc.ListByResourceGroup(az.ctx, applicationResourceGroup)
	if err != nil {
		return "", err
	}

	for _, app := range apps.Values() {
		if appid := *app.ManagedResourceGroupID; appid != "" {
			r, err := az.ResourceGroupFromManagedResourceID(appid)
			if err != nil {
				return "", err
			}
			return r, nil
		}
	}
	return "", nil
}

// ScaleSets returns a slice of VirtualMachineScaleSets within a given resource group
func (az *Client) ScaleSets(resourceGroup string) ([]compute.VirtualMachineScaleSet, error) {
	vmssPages, err := az.ssc.List(az.ctx, resourceGroup)
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
func (az *Client) ScaleSetVMs(resourceGroup string, scaleSet string) ([]compute.VirtualMachineScaleSetVM, error) {
	vmPages, err := az.ssvmc.List(az.ctx, resourceGroup, scaleSet, "", "", "")
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
func (az *Client) UpdateScaleSetsCapacity(resourceGroup string) []error {
	var errs []error
	az.logger.Debugf("listing scale sets for resource group %s", resourceGroup)
	scaleSets, err := az.ScaleSets(resourceGroup)
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	for _, s := range scaleSets {
		az.logger.Debugf("listing virtual machines in scale set %s", *s.Name)
		vms, err := az.ScaleSetVMs(resourceGroup, *s.Name)
		if err != nil {
			errs = append(errs, err)
			return errs
		}
		vmsCount := len(vms)
		az.logger.Debugf("resizing scale set %s from %d to %d virtual machines", *s.Name, vmsCount, vmsCount+1)
		// we only care about possible errors therefore we do not need to process the returned future
		_, err = az.ssc.Update(az.ctx, resourceGroup, *s.Name, compute.VirtualMachineScaleSetUpdate{
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
func (az *Client) UpdateScaleSetsInstanceType(resourceGroup string) []error {
	var errs []error
	az.logger.Debugf("listing scale sets for resource group %s", resourceGroup)
	scaleSets, err := az.ScaleSets(resourceGroup)
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	for _, s := range scaleSets {
		az.logger.Debugf("updating instance type for scale set %s from %s to %s", *s.Name, api.StandardD4sV3, api.StandardD2sV3)
		// we only care about possible errors therefore we do not need to process the returned future
		_, err = az.ssc.Update(az.ctx, resourceGroup, *s.Name, compute.VirtualMachineScaleSetUpdate{
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
func (az *Client) UpdateScaleSetSSHKey(resourceGroup string) []error {
	var errs []error
	var sshKeyData = fakepubkey
	az.logger.Debugf("listing scale sets for resource group %s", resourceGroup)
	scaleSets, err := az.ScaleSets(resourceGroup)
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	for _, s := range scaleSets {
		az.logger.Debugf("updating ssh key for scale set %s", *s.Name)
		// we only care about possible errors therefore we do not need to process the returned future
		_, err := az.ssc.Update(az.ctx, resourceGroup, *s.Name, compute.VirtualMachineScaleSetUpdate{
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
func (az *Client) RebootScaleSetVMs(resourceGroup string) []error {
	var errs []error
	az.logger.Debugf("listing scale sets in resource group %s", resourceGroup)
	scaleSets, err := az.ScaleSets(resourceGroup)
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	for _, s := range scaleSets {
		az.logger.Debugf("listing virtual machines in scale set %s", *s.Name)
		vms, err := az.ScaleSetVMs(resourceGroup, *s.Name)
		if err != nil {
			errs = append(errs, err)
			return errs
		}
		for _, vm := range vms {
			az.logger.Debugf("restarting virtual machine %s", *vm.Name)
			// we only care about possible errors therefore we do not need to process the returned future
			_, err := az.ssvmc.Restart(az.ctx, resourceGroup, *s.Name, *vm.ID)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs
}

// UpdateScaleSetScriptExtension returns a slice of the errors it encounters as it attempts to set script extensions
// for all scale sets within a given resource group
func (az *Client) UpdateScaleSetScriptExtension(resourceGroup string) []error {
	var errs []error
	az.logger.Debugf("listing scale sets in resource group %s", resourceGroup)
	scaleSets, err := az.ScaleSets(resourceGroup)
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	for _, s := range scaleSets {
		az.logger.Debugf("updating script extension for scale set %s", *s.Name)
		// we only care about possible errors therefore we do not need to process the returned future
		_, err := az.ssec.CreateOrUpdate(az.ctx, resourceGroup, *s.Name, "test", compute.VirtualMachineScaleSetExtension{
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

// UpdateCluster updates an OpenshiftManagedCluster by sending both the current external manifest and internal manifest
// which is to be used for the update
func (az *Client) UpdateCluster(external *v20180930preview.OpenShiftManagedCluster, configBlobPath string, config *api.PluginConfig) (*v20180930preview.OpenShiftManagedCluster, error) {
	// Remove the provisioning state before updating
	external.Properties.ProvisioningState = ""

	var oc *v20180930preview.OpenShiftManagedCluster
	var err error
	// simulate the API call to the RP
	if err := wait.PollImmediate(5*time.Second, 1*time.Hour, func() (bool, error) {
		if oc, err = fakerp.CreateOrUpdate(az.ctx, external, az.logger, config); err != nil {
			if autoRestErr, ok := err.(autorest.DetailedError); ok {
				if urlErr, ok := autoRestErr.Original.(*url.Error); ok {
					if netErr, ok := urlErr.Err.(*net.OpError); ok {
						if sysErr, ok := netErr.Err.(*os.SyscallError); ok {
							if sysErr.Err == syscall.ECONNREFUSED {
								return false, nil
							}
						}
					}
				}
			}
			return false, err
		}
		return true, nil
	}); err != nil {
		return nil, err
	}
	return oc, nil
}

type ScaleOperation string

const (
	ScaleOutOperation ScaleOperation = "out"
	ScaleInOperation  ScaleOperation = "in"
)

// ScaleScaleSet scales the capacity of a given scale set to count VMs.
func (az *Client) ScaleScaleSet(scaleSetName string, count int, op ScaleOperation) error {
	var err error
	scaleSet, err := az.ssc.Get(az.ctx, az.resourceGroup, scaleSetName)
	if err != nil {
		az.logger.Errorf("failed to get scale set %s: %v", scaleSetName, err)
		return err
	}
	currentCapacity := *scaleSet.Sku.Capacity
	switch op {
	case ScaleOutOperation:
		if int64(count) < currentCapacity {
			msg := fmt.Sprintf("scale out requires a vm count higher than the current: current=%d, requested=%d", currentCapacity, count)
			az.logger.Errorf(msg)
			err = fmt.Errorf(msg)
			return err
		}
	case ScaleInOperation:
		if int64(count) > currentCapacity {
			msg := fmt.Sprintf("scale in requires a vm count lower than the current: current=%d, requested=%d", currentCapacity, count)
			az.logger.Errorf(msg)
			err = fmt.Errorf(msg)
			return err
		}
	}
	az.logger.Debugf("scaling %s %s scale set to %d vms", op, scaleSetName, count)
	future, err := az.ssc.Update(az.ctx, az.resourceGroup, scaleSetName, compute.VirtualMachineScaleSetUpdate{
		Sku: &compute.Sku{
			Capacity: to.Int64Ptr(int64(count)),
		},
	})
	if err != nil {
		az.logger.Errorf("failed to scale %s scale set %s: %v", op, scaleSetName, err)
		return err
	}
	if err := future.WaitForCompletionRef(az.ctx, az.ssc.Client()); err != nil {
		return err
	}
	return nil
}
