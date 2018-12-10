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
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/updateblob"
	"github.com/openshift/openshift-azure/pkg/jsonpath"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

type ScalesetName string

const updateBlobName = "update"

type UpdateHash interface {
	Initialize(cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}) error
	SetContainer(container storage.Container) error
	Reload() error
	Save() error
	GenerateNewHashes(azuretemplate map[string]interface{}) error
	UpdateInstanceHash(vm *compute.VirtualMachineScaleSetVM)
	DeleteInstanceHash(vmName updateblob.InstanceName)
	FilterOldVMs(vms []compute.VirtualMachineScaleSetVM) ([]compute.VirtualMachineScaleSetVM, error)
	DeleteAllBut(existingVMs map[updateblob.InstanceName]struct{})
	Delete() error
}

type updateHash struct {
	blob            updateblob.Updateblob
	needsSaving     bool
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

// GenerateNewHashes generate the hashes, but don't write them to the blob
func (i *updateHash) GenerateNewHashes(azuretemplate map[string]interface{}) error {
	var err error
	i.ssHashes, err = hashScaleSets(azuretemplate)
	return err
}

// Initialize the blob, assumes this is a deploy and all instances will be the same.
func (i *updateHash) Initialize(cs *api.OpenShiftManagedCluster, azuretemplate map[string]interface{}) error {
	_, err := i.updateContainer.CreateIfNotExists(nil)
	if err != nil {
		return err
	}

	i.ssHashes, err = hashScaleSets(azuretemplate)
	if err != nil {
		return err
	}

	for _, profile := range cs.Properties.AgentPoolProfiles {
		for c := 0; c < profile.Count; c++ {
			name := updateblob.InstanceName(fmt.Sprintf("ss-%s_%d", profile.Name, c))
			i.blob[name] = i.ssHashes[ScalesetName("ss-"+profile.Name)]
		}
	}
	i.needsSaving = true
	return i.Save()
}

func (i *updateHash) UpdateInstanceHash(vm *compute.VirtualMachineScaleSetVM) {
	i.blob[updateblob.InstanceName(*vm.Name)] = i.ssHashes[ssNameForVM(vm)]
	i.needsSaving = true
}

func (i *updateHash) DeleteInstanceHash(vmName updateblob.InstanceName) {
	delete(i.blob, vmName)
	i.needsSaving = true
}

func (i *updateHash) DeleteAllBut(existingVMs map[updateblob.InstanceName]struct{}) {
	for name := range i.blob {
		if _, ok := existingVMs[name]; !ok {
			delete(i.blob, name)
			i.needsSaving = true
		}
	}
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
	if !i.needsSaving {
		return nil
	}
	data, err := json.Marshal(i.blob)
	if err != nil {
		return err
	}

	updateBlob := i.updateContainer.GetBlobReference(updateBlobName)
	err = updateBlob.CreateBlockBlobFromReader(bytes.NewReader(data), nil)
	if err == nil {
		i.needsSaving = false
	}
	return err
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

	i.needsSaving = false
	return nil
}

func hashScaleSets(azuretemplate map[string]interface{}) (map[ScalesetName]updateblob.Hash, error) {
	ssHashes := make(map[ScalesetName]updateblob.Hash)
	for _, r := range jsonpath.MustCompile("$.resources[?(@.type='Microsoft.Compute/virtualMachineScaleSets')]").Get(azuretemplate) {
		original, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		// deep-copy the ARM template since we are mutating it below.
		resource := deepCopy(original)

		// cleanup capacity so that no unnecessary VM rotations are going
		// to occur because of a scale up/down.
		jsonpath.MustCompile("$.sku.capacity").Delete(resource)

		// filter out the nsg dependsOn entry since we remove it
		// during upgrades due to an azure issue.
		jsonpath.MustCompile("$.dependsOn").Delete(resource)

		// hash scale set
		data, err := json.Marshal(resource)
		if err != nil {
			return nil, err
		}
		hf := sha256.New()
		hf.Write(data)

		scaleSetName := jsonpath.MustCompile("$.name").MustGetString(resource)
		ssHashes[ScalesetName(scaleSetName)] = updateblob.Hash(base64.StdEncoding.EncodeToString(hf.Sum(nil)))
	}
	return ssHashes, nil
}

func deepCopy(in map[string]interface{}) map[string]interface{} {
	b, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}
	var out map[string]interface{}
	err = json.Unmarshal(b, &out)
	if err != nil {
		panic(err)
	}
	return out
}
