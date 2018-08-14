package upgrade

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"

	"github.com/openshift/openshift-azure/pkg/api"
)

// NewClientset returns a new Kubernetes typed client.
func NewClientset() (*kubernetes.Clientset, error) {
	restconfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, err
	}
	restconfig.RateLimiter = flowcontrol.NewFakeAlwaysRateLimiter()

	return kubernetes.NewForConfig(restconfig)
}

// VMSSUpgrader is responsible for managing upgrades
// for a VMSS. Both rolling and in-place upgrades are
// supported. Node draining is optional.
type VMSSUpgrader struct {
	SubscriptionID string
	ResourceGroup  string
	// Name of the VMSS to upgrade.
	Name string

	// Plugin provides an interface for configuring
	// waiting for the readiness of a VM.
	Plugin api.Upgrade

	// upgrade parameters
	Drain    bool
	InPlace  bool
	Script   map[string]interface{}
	Count    int64
	ImageRef *compute.ImageReference

	ssc compute.VirtualMachineScaleSetsClient
	vmc compute.VirtualMachineScaleSetVMsClient
	kc  *kubernetes.Clientset
}

func (u *VMSSUpgrader) init() error {
	if err := u.validate(); err != nil {
		return err
	}

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return err
	}

	ssc := compute.NewVirtualMachineScaleSetsClient(u.SubscriptionID)
	ssc.Authorizer = authorizer
	u.ssc = ssc

	vmc := compute.NewVirtualMachineScaleSetVMsClient(u.SubscriptionID)
	vmc.Authorizer = authorizer
	u.vmc = vmc

	kc, err := NewClientset()
	if err != nil {
		return err
	}
	u.kc = kc
	return nil
}

// validate config inputs provided in VMSSUpgrader.
func (u *VMSSUpgrader) validate() error {
	if u.SubscriptionID == "" {
		return errors.New("Azure subscription id is required")
	}
	if u.ResourceGroup == "" {
		return errors.New("Azure resource group is required")
	}
	if u.Name == "" {
		return errors.New("VMSS name is required")
	}
	if u.ImageRef != nil &&
		(u.ImageRef.ID != nil && *u.ImageRef.ID != "") &&
		((u.ImageRef.Sku != nil && *u.ImageRef.Sku != "") || (u.ImageRef.Version != nil && *u.ImageRef.Version != "")) {
		return errors.New("imageReference.id cannot be used with imageReference.version and imageReference.sku")
	}
	if u.Count < 1 {
		return errors.New("replica count needs to be a positive number")
	}
	return nil
}

// Upgrade updates a VMSS and subsequently upgrades
// all VMs in the VMSS, via either rolling them one
// by one or in-place update. The VMSS is assumed to
// use manual rollingUpgrade policy. Safeguards are
// in place for ensuring the minimum available VMs
// are always up and running (that is u.Count in case
// of a rolling upgrade and u.Count-1 in case of an
// in-place upgrade.
func (u *VMSSUpgrader) Upgrade() error {
	if err := u.init(); err != nil {
		return err
	}

	vmss, err := u.ssc.Get(context.Background(), u.ResourceGroup, u.Name)
	if err != nil {
		return fmt.Errorf("cannot get VMSS %q in resource group %q: %v", u.Name, u.ResourceGroup, err)
	}

	if vmss, err = u.updateVMSS(vmss); err != nil {
		return err
	}

	log.Infof("VMSS %q current capacity at %d", u.Name, *vmss.Sku.Capacity)

	vmsToUpgrade, err := u.upgradable()
	if err != nil {
		return fmt.Errorf("cannot list upgradable VMs in VMSS %q: %v", u.Name, err)
	}

	for _, vmToUpgrade := range vmsToUpgrade {
		canScaleUp, err := u.canScaleUp()
		if err != nil {
			return err
		}
		if canScaleUp {
			log.Infof("Scale up VMSS %q", u.Name)
			*vmss.Sku.Capacity = u.Count + 1
			res, err := u.ssc.CreateOrUpdate(context.Background(), u.ResourceGroup, u.Name, vmss)
			if err != nil {
				return fmt.Errorf("cannot scale up VMSS %q: %v", u.Name, err)
			}
			if err := res.WaitForCompletion(context.Background(), u.ssc.BaseClient.Client); err != nil {
				return fmt.Errorf("cannot wait for VMSS %q scale up: %v", u.Name, err)
			}
			if vmss, err = res.Result(u.ssc); err != nil {
				return fmt.Errorf("error during VMSS %q scale up: %v", u.Name, err)
			}
		}

		if err := u.canScaleDown(); err != nil {
			return err
		}

		// Before we can delete the node we should safely and responsibly drain it
		if u.Drain {
			log.Infof("Drain VM %q in VMSS %q", *vmToUpgrade.InstanceID, u.Name)
			nodeName := *vmToUpgrade.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName
			if err := u.drain(nodeName); err != nil {
				return err
			}
		}

		// Re-image VMs only on in-place upgrades. Otherwise, we have already
		// scaled up and need to scale down old VMs.
		if u.InPlace && u.reimageOnly() {
			log.Infof("Reimage VM %q in VMSS %q", *vmToUpgrade.InstanceID, u.Name)
			update := compute.VirtualMachineScaleSetVMInstanceIDs{
				InstanceIds: &[]string{*vmToUpgrade.InstanceID},
			}
			res, err := u.ssc.Reimage(context.Background(), u.ResourceGroup, u.Name, &update)
			if err != nil {
				return fmt.Errorf("cannot update VM %q: %v", *vmToUpgrade.InstanceID, err)
			}
			if err := res.WaitForCompletion(context.Background(), u.vmc.BaseClient.Client); err != nil {
				return fmt.Errorf("cannot wait for VM %q reimage: %v", *vmToUpgrade.InstanceID, err)
			}
			if _, err := res.Result(u.ssc); err != nil {
				return fmt.Errorf("error during VM %q reimage: %v", *vmToUpgrade.InstanceID, err)
			}
			// Wait for VM readiness; this may vary between VMSSs, eg. a master VM
			// is ready when the API server pod running on it is ready whereas a compute
			// VM is ready when it has joined the cluster as a Ready k8s node.
			nodeName := *vmToUpgrade.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName
			if err := u.waitForReady(nodeName); err != nil {
				return err
			}
			if u.Drain {
				if err := u.uncordon(nodeName); err != nil {
					return err
				}
			}
			continue
		}

		if u.InPlace {
			log.Infof("In-place upgrade VM %q in VMSS %q", *vmToUpgrade.InstanceID, u.Name)
			update := compute.VirtualMachineScaleSetVMInstanceRequiredIDs{
				InstanceIds: &[]string{*vmToUpgrade.InstanceID},
			}
			res, err := u.ssc.UpdateInstances(context.Background(), u.ResourceGroup, u.Name, update)
			if err != nil {
				return fmt.Errorf("cannot update VM %q: %v", *vmToUpgrade.InstanceID, err)
			}
			if err := res.WaitForCompletion(context.Background(), u.vmc.BaseClient.Client); err != nil {
				return fmt.Errorf("cannot wait for VM %q update: %v", *vmToUpgrade.InstanceID, err)
			}
			if _, err := res.Result(u.ssc); err != nil {
				return fmt.Errorf("error during VM %q update: %v", *vmToUpgrade.InstanceID, err)
			}
			// Wait for VM readiness; this may vary between VMSSs, eg. a master VM
			// is ready when the API server pod running on it is ready whereas a compute
			// VM is ready when it has joined the cluster as a Ready k8s node.
			nodeName := *vmToUpgrade.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName
			if err := u.waitForReady(nodeName); err != nil {
				return err
			}
			if u.Drain {
				if err := u.uncordon(nodeName); err != nil {
					return err
				}
			}
			continue
		}

		log.Infof("Deleting VM %q in VMSS %q", *vmToUpgrade.InstanceID, u.Name)
		if _, err := u.vmc.Delete(context.Background(), u.ResourceGroup, u.Name, *vmToUpgrade.InstanceID); err != nil {
			return fmt.Errorf("cannot delete VM %q: %v", *vmToUpgrade.InstanceID, err)
		}
	}

	log.Infof("Completed upgrading VMSS %q", u.Name)
	return nil
}

// updateVMSS is meant to update the VMSS to the desired state before
// manually rolling the VMs.
func (u *VMSSUpgrader) updateVMSS(vmss compute.VirtualMachineScaleSet) (compute.VirtualMachineScaleSet, error) {
	if u.reimageOnly() {
		return vmss, nil
	}

	if u.ImageRef != nil {
		imageRef := vmss.VirtualMachineScaleSetProperties.VirtualMachineProfile.StorageProfile.ImageReference
		if u.ImageRef.ID != nil {
			imageRef.ID = u.ImageRef.ID
		} else {
			imageRef.Sku = u.ImageRef.Sku
			imageRef.Version = u.ImageRef.Version
		}
	}

	if u.Script != nil {
		extensions := *vmss.VirtualMachineScaleSetProperties.VirtualMachineProfile.ExtensionProfile.Extensions
		for i, ext := range extensions {
			if ext.Name != nil && *ext.Name == "cse" {
				extensions[i].ProtectedSettings = u.Script
			}
		}
	}

	log.Infof("Updating VMSS %q ...", u.Name)
	res, err := u.ssc.CreateOrUpdate(context.Background(), u.ResourceGroup, u.Name, vmss)
	if err != nil {
		return vmss, fmt.Errorf("cannot update VMSS %q: %v", u.Name, err)
	}
	if err := res.WaitForCompletion(context.Background(), u.ssc.BaseClient.Client); err != nil {
		return vmss, fmt.Errorf("cannot wait for VMSS %q update: %v", u.Name, err)
	}
	if vmss, err = res.Result(u.ssc); err != nil {
		return vmss, fmt.Errorf("error during VMSS %q update: %v", u.Name, err)
	}
	return vmss, nil
}

// upgradable returns all VMs that need to be upgraded.
func (u *VMSSUpgrader) upgradable() ([]compute.VirtualMachineScaleSetVM, error) {
	vmList, err := u.vmc.List(context.Background(), u.ResourceGroup, u.Name, "", "", "")
	if err != nil {
		return nil, err
	}

	var needUpgrade []compute.VirtualMachineScaleSetVM
	for _, vm := range vmList.Values() {
		if u.needsUpgrade(&vm) {
			needUpgrade = append(needUpgrade, vm)
		}
	}

	return needUpgrade, nil
}

// reimageOnly is meant to return true when there are no config inputs
// provided to the upgrade process so instead of an in-place update, a
// reimage of all the VMs is triggered.
func (u *VMSSUpgrader) reimageOnly() bool {
	return u.ImageRef == nil && u.Script == nil
}

func (u *VMSSUpgrader) needsUpgrade(vm *compute.VirtualMachineScaleSetVM) bool {
	if u.reimageOnly() {
		// If no imageRef config has been passed, perform a re-image of all VMs.
		return true
	}

	vmProps := vm.VirtualMachineScaleSetVMProperties
	if vmProps == nil || vmProps.StorageProfile == nil || vmProps.StorageProfile.ImageReference == nil {
		log.Warn("If you see this, search for the break")
		return false
	}

	imageRef := vmProps.StorageProfile.ImageReference
	return u.Script != nil || u.isImageUpdate(imageRef)
}

func (u *VMSSUpgrader) isImageUpdate(imageRef *compute.ImageReference) bool {
	if u.ImageRef == nil {
		return false
	}
	return ((imageRef.Version != nil && u.ImageRef.Version != nil && *imageRef.Version != *u.ImageRef.Version) ||
		(imageRef.Sku != nil && u.ImageRef.Sku != nil && *imageRef.Sku != *u.ImageRef.Sku)) ||
		(imageRef.ID != nil && u.ImageRef.ID != nil && *imageRef.ID != *u.ImageRef.ID)
}

// canScaleUp returns whether the upgrade process can scale up the VMSS.
// Scaling up is ignored in case of an in-place upgrade.
func (u *VMSSUpgrader) canScaleUp() (bool, error) {
	// Do not scale up on in-place upgrades.
	if u.InPlace {
		return false, nil
	}
	vmss, err := u.ssc.Get(context.Background(), u.ResourceGroup, u.Name)
	if err != nil {
		return false, fmt.Errorf("cannot get VMSS %q in resource group %q: %v", u.Name, u.ResourceGroup, err)
	}
	log.Debugf("VMSS %q scale up: maximum capacity at %d, current count at %d", u.Name, u.Count+1, *vmss.Sku.Capacity)
	return *vmss.Sku.Capacity < u.Count+1, nil
}

// canScaleDown waits until all VMs in a scale set are healthy.
func (u *VMSSUpgrader) canScaleDown() error {
	log.Infof("Waiting for healthy VMs for VMSS %q", u.Name)
	return wait.PollImmediate(5*time.Second, 20*time.Minute, func() (bool, error) {
		vmss, err := u.ssc.Get(context.Background(), u.ResourceGroup, u.Name)
		if err != nil {
			return false, err
		}
		vmList, err := u.vmc.List(context.Background(), u.ResourceGroup, u.Name, "", "", "")
		if err != nil {
			return false, err
		}

		healthy := int64(0)
		for _, vm := range vmList.Values() {
			// Wait for VM readiness; this may vary between VMSSs, eg. a master VM
			// is ready when the API server pod running on it is ready whereas a compute
			// VM is ready when it has joined the cluster as a Ready k8s node.
			nodeName := *vm.VirtualMachineScaleSetVMProperties.OsProfile.ComputerName
			isReady, err := u.Plugin.IsReady(nodeName)
			if err != nil {
				return false, err
			}
			if isReady {
				healthy++
			}
		}

		log.Debugf("VMSS %q scale down: healthy VMs at %d, current count at %d", u.Name, healthy, *vmss.Sku.Capacity)
		return *vmss.Sku.Capacity == healthy, nil
	})
}

// waitForReady waits until the provided node is considered ready.
// Node readiness may vary between scale sets, thus IsReady is abstracted
// into its own interface.
func (u *VMSSUpgrader) waitForReady(nodeName string) error {
	return wait.PollImmediate(2*time.Second, 10*time.Minute, func() (bool, error) {
		return u.Plugin.IsReady(nodeName)
	})
}
