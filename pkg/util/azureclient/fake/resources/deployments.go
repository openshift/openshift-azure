package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	azcompute "github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/Azure/go-autorest/autorest"

	"github.com/openshift/openshift-azure/pkg/util/azureclient/compute"
	fakestorage "github.com/openshift/openshift-azure/pkg/util/azureclient/fake/storage"
)

// FakeDeploymentsClient is a Fake of DeploymentsClient interface
type FakeDeploymentsClient struct {
	ssc compute.VirtualMachineScaleSetsClient
	srp *fakestorage.StorageRP
}

// NewFakeDeploymentsClient creates a new Fake instance
func NewFakeDeploymentsClient(ssc compute.VirtualMachineScaleSetsClient, srp *fakestorage.StorageRP) *FakeDeploymentsClient {
	return &FakeDeploymentsClient{ssc: ssc, srp: srp}
}

func (d *FakeDeploymentsClient) CreateOrUpdate(ctx context.Context, resourceGroupName string, deploymentName string, parameters resources.Deployment) (resources.DeploymentsCreateOrUpdateFuture, error) {
	return resources.DeploymentsCreateOrUpdateFuture{}, fmt.Errorf("not implemented")
}

type fakeClient struct {
}

func (f *fakeClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200}, nil
}

func (d *FakeDeploymentsClient) Client() autorest.Client {
	return autorest.Client{Sender: &fakeClient{}}
}

// CreateOrUpdate Fakes base method
// store in memory the resources that are created so that other api requests can work with them
func (d *FakeDeploymentsClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName string, deploymentName string, parameters resources.Deployment) error {
	templ := parameters.Properties.Template.(map[string]interface{})
	for _, r := range templ["resources"].([]interface{}) {
		rMap := r.(map[string]interface{})
		if strings.Contains(rMap["type"].(string), "Microsoft.Storage/storageAccounts") {
			sa := storage.Account{}
			var sab []byte
			sab, err := json.Marshal(rMap)
			if err != nil {
				return err
			}
			err = sa.UnmarshalJSON(sab)
			if err != nil {
				return err
			}

			updated := false
			for a, acct := range d.srp.Accts {
				if acct.Name == sa.Name {
					d.srp.Accts[a] = sa
					updated = true
				}
			}
			if !updated {
				d.srp.Accts = append(d.srp.Accts, sa)
			}
		} else if strings.Contains(rMap["type"].(string), "Microsoft.Compute/virtualMachineScaleSets") {
			vm := azcompute.VirtualMachineScaleSet{}
			rb, err := json.Marshal(rMap)
			if err != nil {
				return err
			}
			err = vm.UnmarshalJSON(rb)
			if err != nil {
				return err
			}
			err = d.ssc.CreateOrUpdate(ctx, resourceGroupName, *vm.Name, vm)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
