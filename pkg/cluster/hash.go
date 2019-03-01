package cluster

//go:generate go get github.com/golang/mock/gomock
//go:generate go install github.com/golang/mock/mockgen
//go:generate mockgen -destination=../util/mocks/mock_$GOPACKAGE/hash.go -package=mock_$GOPACKAGE -source hash.go
//go:generate gofmt -s -l -w ../util/mocks/mock_$GOPACKAGE/hash.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../util/mocks/mock_$GOPACKAGE/hash.go

import (
	"crypto/sha256"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/arm"
)

type Hasher interface {
	HashScaleSet(*api.OpenShiftManagedCluster, *api.AgentPoolProfile) ([]byte, error)
}

type hasher struct {
	testConfig api.TestConfig
}

func hashVMSS(vmss *compute.VirtualMachineScaleSet) ([]byte, error) {
	data, err := json.Marshal(vmss)
	if err != nil {
		return nil, err
	}

	hf := sha256.New()
	hf.Write(data)

	return hf.Sum(nil), nil
}

// hashScaleSets returns the set of desired state scale set hashes
func (h *hasher) HashScaleSet(cs *api.OpenShiftManagedCluster, app *api.AgentPoolProfile) ([]byte, error) {
	// the hash is invariant of name, suffix, count
	appCopy := *app
	appCopy.Count = 0
	appCopy.Name = ""

	vmss, err := arm.Vmss(cs, &appCopy, "", "", h.testConfig) // TODO: backupBlob is rather a layering violation here
	if err != nil {
		return nil, err
	}

	return hashVMSS(vmss)
}
