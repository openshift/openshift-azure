package fake

import (
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	azurestorage "github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

type AzureCloud struct {
	log     *logrus.Entry
	Vms     []compute.VirtualMachineScaleSetVM
	Ssc     []compute.VirtualMachineScaleSet
	Secrets []keyvault.SecretBundle
	Accts   []storage.Account
	Blobs   map[string]map[string][]byte

	AccountsClient                  azureclient.AccountsClient
	StorageClient                   azurestorage.Client
	DeploymentsClient               azureclient.DeploymentsClient
	KeyVaultClient                  azureclient.KeyVaultClient
	VirtualMachineScaleSetVMsClient azureclient.VirtualMachineScaleSetVMsClient
	VirtualMachineScaleSetsClient   azureclient.VirtualMachineScaleSetsClient
}

func NewFakeAzureCloud(log *logrus.Entry, vms []compute.VirtualMachineScaleSetVM, ssc []compute.VirtualMachineScaleSet, secrets []keyvault.SecretBundle, accts []storage.Account, blobs map[string]map[string][]byte) *AzureCloud {
	az := &AzureCloud{
		log:     log,
		Vms:     vms,
		Ssc:     ssc,
		Secrets: secrets,
		Accts:   accts,
		Blobs:   blobs,
	}
	az.AccountsClient = NewFakeAccountsClient(az)
	az.StorageClient = NewFakeStorageClient(az)
	az.KeyVaultClient = NewFakeKeyVaultClient(az)
	az.DeploymentsClient = NewFakeDeploymentsClient(az)
	az.VirtualMachineScaleSetVMsClient = NewFakeVirtualMachineScaleSetVMsClient(az)
	az.VirtualMachineScaleSetsClient = NewFakeVirtualMachineScaleSetsClient(az)
	return az
}

type fakeClient struct {
}

func (f *fakeClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200}, nil
}

func allwaysDoneClient() autorest.Client {
	return autorest.Client{Sender: &fakeClient{}}
}
