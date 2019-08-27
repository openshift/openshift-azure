package privatelink

import (
	"context"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

// PrivateLinkServicesClient is the network Client
type PrivateLinkServicesClient struct {
	network.BaseClient
}

// NewPrivateLinkServicesClient creates an instance of the PrivateLinkServicesClient client.
func NewPrivateLinkServicesClient(subscriptionID string) PrivateLinkServicesClient {
	return NewNewPrivateLinkServicesClientWithBaseURI(DefaultBaseURI, subscriptionID)
}

// NewNewPrivateLinkServicesClientWithBaseURI creates an instance of the PrivateLinkServicesClient client.
func NewNewPrivateLinkServicesClientWithBaseURI(baseURI string, subscriptionID string) PrivateLinkServicesClient {
	return PrivateLinkServicesClient{NewWithBaseURI(baseURI, subscriptionID)}
}

// Get gets the specified private link service by resource group.
// Parameters:
// resourceGroupName - the name of the resource group.
// serviceName - the name of the private link service.
// expand - expands referenced resources.
func (client PrivateLinkServicesClient) Get(ctx context.Context, resourceGroupName string, serviceName string, expand string) (result PrivateLinkService, err error) {
	req, err := client.GetPreparer(ctx, resourceGroupName, serviceName, expand)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.PrivateLinkServicesClient", "Get", nil, "Failure preparing request")
		return
	}

	resp, err := client.GetSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "network.PrivateLinkServicesClient", "Get", resp, "Failure sending request")
		return
	}

	result, err = client.GetResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.PrivateLinkServicesClient", "Get", resp, "Failure responding to request")
	}

	return
}

// GetPreparer prepares the Get request.
func (client PrivateLinkServicesClient) GetPreparer(ctx context.Context, resourceGroupName string, serviceName string, expand string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"serviceName":       autorest.Encode("path", serviceName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2019-06-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}
	if len(expand) > 0 {
		queryParameters["$expand"] = autorest.Encode("query", expand)
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateLinkServices/{serviceName}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// GetSender sends the Get request. The method will close the
// http.Response Body if it receives an error.
func (client PrivateLinkServicesClient) GetSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req, azure.DoRetryWithRegistration(client.Client))
}

// GetResponder handles the response to the Get request. The method always
// closes the http.Response Body.
func (client PrivateLinkServicesClient) GetResponder(resp *http.Response) (result PrivateLinkService, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}
