package fake

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	azurestorage "github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

type ComputeRP struct {
	Vms map[string][]compute.VirtualMachineScaleSetVM
	Ssc []compute.VirtualMachineScaleSet
}

type VaultRP struct {
	Secrets []keyvault.SecretBundle
}

type StorageRP struct {
	Accts []storage.Account
	Blobs map[string]map[string][]byte
}

type AzureCloud struct {
	ComputeRP
	StorageRP
	VaultRP

	log *logrus.Entry

	AccountsClient                  azureclient.AccountsClient
	StorageClient                   azurestorage.Client
	DeploymentsClient               azureclient.DeploymentsClient
	KeyVaultClient                  azureclient.KeyVaultClient
	VirtualMachineScaleSetVMsClient azureclient.VirtualMachineScaleSetVMsClient
	VirtualMachineScaleSetsClient   azureclient.VirtualMachineScaleSetsClient
}

func NewFakeAzureCloud(log *logrus.Entry, secrets []keyvault.SecretBundle) *AzureCloud {
	az := &AzureCloud{
		log: log,
		ComputeRP: ComputeRP{
			Vms: map[string][]compute.VirtualMachineScaleSetVM{},
			Ssc: []compute.VirtualMachineScaleSet{},
		},
		VaultRP: VaultRP{Secrets: secrets},
		StorageRP: StorageRP{
			Accts: []storage.Account{},
			Blobs: map[string]map[string][]byte{},
		},
	}
	az.AccountsClient = NewFakeAccountsClient(az)
	az.StorageClient = NewFakeStorageClient(az)
	az.KeyVaultClient = NewFakeKeyVaultClient(az)
	az.DeploymentsClient = NewFakeDeploymentsClient(az)
	az.VirtualMachineScaleSetVMsClient = NewFakeVirtualMachineScaleSetVMsClient(az)
	az.VirtualMachineScaleSetsClient = NewFakeVirtualMachineScaleSetsClient(az)
	return az
}
