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
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/addons"
	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/arm"
	"github.com/openshift/openshift-azure/pkg/config"
	"github.com/openshift/openshift-azure/pkg/util/writers"
)

type Hasher interface {
	HashWorkerScaleSet(*api.OpenShiftManagedCluster, *api.AgentPoolProfile) ([]byte, error)
	HashMasterScaleSet(cs *api.OpenShiftManagedCluster, app *api.AgentPoolProfile, instanceID int64) ([]byte, error)
	HashSyncPod(cs *api.OpenShiftManagedCluster) ([]byte, error)
}

type hasher struct {
	log        *logrus.Entry
	testConfig api.TestConfig
}

func (h *hasher) vmss(cs *api.OpenShiftManagedCluster, app *api.AgentPoolProfile) (*compute.VirtualMachineScaleSet, error) {
	// the hash is invariant of name, suffix, count
	appCopy := *app
	appCopy.Count = 0
	appCopy.Name = ""

	return arm.Vmss(cs, &appCopy, "", "", h.testConfig) // TODO: backupBlob is rather a layering violation here
}

// HashWorkerScaleSet returns the hash of a worker scale set
func (h *hasher) HashWorkerScaleSet(cs *api.OpenShiftManagedCluster, app *api.AgentPoolProfile) ([]byte, error) {
	hash := sha256.New()

	vmss, err := h.vmss(cs, app)
	if err != nil {
		return nil, err
	}

	err = json.NewEncoder(hash).Encode(vmss)
	if err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

// HashMasterScaleSet returns the hash of a worker scale set
func (h *hasher) HashMasterScaleSet(cs *api.OpenShiftManagedCluster, app *api.AgentPoolProfile, instanceID int64) ([]byte, error) {
	hash := sha256.New()

	vmss, err := h.vmss(cs, app)
	if err != nil {
		return nil, err
	}

	err = json.NewEncoder(hash).Encode(vmss)
	if err != nil {
		return nil, err
	}

	err = arm.WriteStartupFiles(h.log, cs, app.Role, writers.NewTarWriter(hash), config.GetHostname(app, "", instanceID), "")
	if err != nil {
		return nil, err
	}

	// add certificates pulled from keyvault by the master to the hash, to
	// ensure the masters update if a cert changes.  We don't add the keys
	// because these are not necessarily stable (sometimes the 'D' value of
	// the RSA key returned by keyvault differs to the one that was sent).
	// I believe that in a given RSA key, there are multiple suitable values
	// of 'D', so this is not a problem, however it doesn't make the value
	// suitable for a hash.  References:
	// https://stackoverflow.com/a/14233140,
	// https://crypto.stackexchange.com/a/46572.
	hash.Write(cs.Config.Certificates.OpenShiftConsole.Certs[0].Raw)
	hash.Write(cs.Config.Certificates.Router.Certs[0].Raw)

	return hash.Sum(nil), nil
}

// HashSyncPod returns the hash of the sync pod output
func (h *hasher) HashSyncPod(cs *api.OpenShiftManagedCluster) ([]byte, error) {
	hash := sha256.New()

	m, err := addons.ReadDB(cs)
	if err != nil {
		return nil, err
	}

	err = json.NewEncoder(hash).Encode(m)
	if err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}
