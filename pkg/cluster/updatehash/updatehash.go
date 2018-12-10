package updatehash

//go:generate go get github.com/golang/mock/gomock
//go:generate go install github.com/golang/mock/mockgen
//go:generate mockgen -destination=../../util/mocks/mock_$GOPACKAGE/instancehash.go github.com/openshift/openshift-azure/pkg/cluster/$GOPACKAGE UpdateHash
//go:generate gofmt -s -l -w ../../util/mocks/mock_$GOPACKAGE/instancehash.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../util/mocks/mock_$GOPACKAGE/instancehash.go

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

type ScalesetName string

const updateBlobName = "update"

// UpdateHash contains the actions required for the instance hashes needed by reentrant updates
type UpdateHash interface {
	Initialize(cs *api.OpenShiftManagedCluster) error
	SetContainer(container storage.Container) error
	Reload() error
	Save() error
	HashScaleSets(pluginConfig api.PluginConfig, cs *api.OpenShiftManagedCluster) error
	UpdateInstanceHash(vm *compute.VirtualMachineScaleSetVM) error
	DeleteInstanceHash(vmName updateblob.InstanceName) error
	FilterOldVMs(vms []compute.VirtualMachineScaleSetVM) ([]compute.VirtualMachineScaleSetVM, error)
	DeleteAllBut(existingVMs map[updateblob.InstanceName]struct{}) error
	Delete() error
}

type updateHash struct {
	blob            updateblob.Updateblob
	ssHashes        map[ScalesetName]updateblob.Hash
	updateContainer storage.Container
	log             *logrus.Entry
}

func NewUpdateHash(log *logrus.Entry) UpdateHash {
	return &updateHash{
		log:  log,
		blob: updateblob.Updateblob{},
	}
}

func ssNameForVM(vm *compute.VirtualMachineScaleSetVM) ScalesetName {
	hostname := strings.Split(*vm.Name, "_")[0]
	return ScalesetName(hostname)
}

func (i *updateHash) SetContainer(container storage.Container) error {
	i.updateContainer = container
	_, err := i.updateContainer.CreateIfNotExists(nil)
	return err
}

// Initialize the blob, assumes this is a deploy and all instances will be the same.
func (i *updateHash) Initialize(cs *api.OpenShiftManagedCluster) error {
	_, err := i.updateContainer.CreateIfNotExists(nil)
	if err != nil {
		return err
	}

	for _, app := range cs.Properties.AgentPoolProfiles {
		for c := 0; c < app.Count; c++ {
			name := updateblob.InstanceName(config.GetInstanceName(app.Name, c))
			i.blob[name] = i.ssHashes[ScalesetName(config.GetScalesetName(app.Name))]
		}
	}
	return i.Save()
}

func (i *updateHash) UpdateInstanceHash(vm *compute.VirtualMachineScaleSetVM) error {
	i.blob[updateblob.InstanceName(*vm.Name)] = i.ssHashes[ssNameForVM(vm)]
	return i.Save()
}

func (i *updateHash) DeleteInstanceHash(vmName updateblob.InstanceName) error {
	delete(i.blob, vmName)
	return i.Save()
}

func (i *updateHash) DeleteAllBut(existingVMs map[updateblob.InstanceName]struct{}) error {
	needsSaving := false
	for name := range i.blob {
		if _, ok := existingVMs[name]; !ok {
			delete(i.blob, name)
			needsSaving = true
		}
	}
	if needsSaving {
		return i.Save()
	}
	return nil
}

func (i *updateHash) FilterOldVMs(vms []compute.VirtualMachineScaleSetVM) ([]compute.VirtualMachineScaleSetVM, error) {
	err := i.Reload()
	if err != nil {
		return nil, err
	}
	var oldVMs []compute.VirtualMachineScaleSetVM
	for _, vm := range vms {
		if i.blob[updateblob.InstanceName(*vm.Name)] != i.ssHashes[ssNameForVM(&vm)] {
			oldVMs = append(oldVMs, vm)
		} else {
			i.log.Infof("skipping vm %q since it's already updated", *vm.Name)
		}
	}
	return oldVMs, nil
}

func (i *updateHash) Delete() error {
	bc := i.updateContainer.GetBlobReference(updateBlobName)
	return bc.Delete(nil)
}

func (i *updateHash) Save() error {
	data, err := json.Marshal(i.blob)
	if err != nil {
		return err
	}

	updateBlob := i.updateContainer.GetBlobReference(updateBlobName)
	return updateBlob.CreateBlockBlobFromReader(bytes.NewReader(data), nil)
}

func (i *updateHash) Reload() error {
	updateBlob := i.updateContainer.GetBlobReference(updateBlobName)
	rc, err := updateBlob.Get(nil)
	if err != nil {
		return err
	}
	defer rc.Close()

	d := json.NewDecoder(rc)

	if err := d.Decode(&i.blob); err != nil {
		return err
	}

	return nil
}

func (i *updateHash) HashScaleSets(pluginConfig api.PluginConfig, cs *api.OpenShiftManagedCluster) error {
	for _, app := range cs.Properties.AgentPoolProfiles {
		// TODO: backupBlob is rather a layering violation here
		// IMO(Angus) it's only because we now have to keep re-generating the arm.
		// We used to generate the arm once and pass it around, any reason why
		// we can't do that?
		vmss, err := arm.Vmss(&pluginConfig, cs, &app, "")
		if err != nil {
			return err
		}

		err = i.hashVMSS(vmss)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *updateHash) hashVMSS(vmss *compute.VirtualMachineScaleSet) error {
	// cleanup capacity so that no unnecessary VM rotations are going to occur
	// because of a scale up/down.
	if vmss.Sku != nil {
		vmss.Sku.Capacity = nil
	}

	data, err := json.Marshal(vmss)
	if err != nil {
		return err
	}

	hf := sha256.New()
	hf.Write(data)

	i.ssHashes[ScalesetName(*vmss.Name)] = updateblob.Hash(base64.StdEncoding.EncodeToString(hf.Sum(nil)))
	return nil
}
