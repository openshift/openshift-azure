package fake

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2018-02-01/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

// FakeDeploymentsClient is a Fake of DeploymentsClient interface
type FakeDeploymentsClient struct {
	az *AzureCloud
}

// NewFakeDeploymentsClient creates a new Fake instance
func NewFakeDeploymentsClient(az *AzureCloud) *FakeDeploymentsClient {
	return &FakeDeploymentsClient{az: az}
}

// Client Fakes base method
func (d *FakeDeploymentsClient) Client() autorest.Client {
	return allwaysDoneClient()
}

// CreateOrUpdate Fakes base method
func (d *FakeDeploymentsClient) CreateOrUpdate(ctx context.Context, resourceGroupName string, deploymentName string, parameters resources.Deployment) (result resources.DeploymentsCreateOrUpdateFuture, err error) {
	var testURL *url.URL
	testURL, err = url.Parse("https://example.com/nothing")

	result.Future, _ = azure.NewFutureFromResponse(&http.Response{
		StatusCode: 200,
		Request: &http.Request{
			Method: http.MethodPut,
			URL:    testURL,
		},
	})
	templ := parameters.Properties.Template.(map[string]interface{})
	for _, r := range templ["resources"].([]interface{}) {
		rMap := r.(map[string]interface{})
		if strings.Contains(rMap["type"].(string), "Microsoft.Storage/storageAccounts") {
			sa := storage.Account{}
			var sab []byte
			sab, err = json.Marshal(rMap)
			if err != nil {
				result.Future.Response().StatusCode = 500
				return
			}
			err = sa.UnmarshalJSON(sab)
			if err != nil {
				result.Future.Response().StatusCode = 500
				return
			}

			updated := false
			for a, acct := range d.az.Accts {
				if acct.Name == sa.Name {
					d.az.Accts[a] = sa
					updated = true
				}
			}
			if !updated {
				d.az.Accts = append(d.az.Accts, sa)
			}
		} else if strings.Contains(rMap["type"].(string), "Microsoft.Compute/virtualMachineScaleSets") {
			vm := compute.VirtualMachineScaleSet{}
			rb, _ := json.Marshal(rMap)
			vm.UnmarshalJSON(rb)
			d.az.VirtualMachineScaleSetsClient.CreateOrUpdate(ctx, resourceGroupName, *vm.Name, vm)
		}
	}

	// store in memory the resources that are created so that other api requests can work with them
	err = nil
	return
}
