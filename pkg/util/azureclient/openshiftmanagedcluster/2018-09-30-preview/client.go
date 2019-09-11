package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"

	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview"
)

const (
	// DefaultBaseURI is the default URI used for the service Containerservice
	DefaultBaseURI = "https://management.azure.com"
)

// BaseClient is the base client for Containerservice.
type BaseClient struct {
	autorest.Client
	BaseURI        string
	SubscriptionID string
}

// New creates an instance of the BaseClient client.
func New(subscriptionID string) BaseClient {
	return NewWithBaseURI(DefaultBaseURI, subscriptionID)
}

// NewWithBaseURI creates an instance of the BaseClient client.
func NewWithBaseURI(baseURI string, subscriptionID string) BaseClient {
	return BaseClient{
		Client:         autorest.NewClientWithUserAgent(UserAgent()),
		BaseURI:        baseURI,
		SubscriptionID: subscriptionID,
	}
}

// UserAgent returns the UserAgent string to use when sending http.Requests.
func UserAgent() string {
	return "openshift-azure/pkg/api/2018-09-30-preview/api/client"
}

// Version returns the semantic version (see http://semver.org) of the client.
func Version() string {
	return "0.0.0"
}

// TagsObject tags object for patch operations.
type TagsObject struct {
	// Tags - Resource tags.
	Tags map[string]*string `json:"tags"`
}

// MarshalJSON is the custom marshaler for TagsObject.
func (toVar TagsObject) MarshalJSON() ([]byte, error) {
	objectMap := make(map[string]interface{})
	if toVar.Tags != nil {
		objectMap["tags"] = toVar.Tags
	}
	return json.Marshal(objectMap)
}

// OpenShiftManagedClustersUpdateTagsFuture an abstraction for monitoring and retrieving the results of a
// long-running operation.
type OpenShiftManagedClustersUpdateTagsFuture struct {
	azure.Future
}

// Result returns the result of the asynchronous operation.
// If the operation has not completed it will return an error.
func (future *OpenShiftManagedClustersUpdateTagsFuture) Result(client OpenShiftManagedClustersClient) (osmc v20180930preview.OpenShiftManagedCluster, err error) {
	var done bool
	done, err = future.DoneWithContext(context.Background(), client)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersUpdateTagsFuture", "Result", future.Response(), "Polling failure")
		return
	}
	if !done {
		err = azure.NewAsyncOpIncompleteError("containerservice.OpenShiftManagedClustersUpdateTagsFuture")
		return
	}
	sender := autorest.DecorateSender(client, autorest.DoRetryForStatusCodes(client.RetryAttempts, client.RetryDuration, autorest.StatusCodesForRetry...))
	if osmc.Response.Response, err = future.GetResult(sender); err == nil && osmc.Response.Response.StatusCode != http.StatusNoContent {
		osmc, err = client.UpdateTagsResponder(osmc.Response.Response)
		if err != nil {
			err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersUpdateTagsFuture", "Result", osmc.Response.Response, "Failure responding to request")
		}
	}
	return
}

// OpenShiftManagedClustersClient is the the Container Service Client.
type OpenShiftManagedClustersClient struct {
	BaseClient
}

// NewOpenShiftManagedClustersClient creates an instance of the OpenShiftManagedClustersClient client.
func NewOpenShiftManagedClustersClient(subscriptionID string) OpenShiftManagedClustersClient {
	return NewOpenShiftManagedClustersClientWithBaseURI(DefaultBaseURI, subscriptionID)
}

// NewOpenShiftManagedClustersClientWithBaseURI creates an instance of the OpenShiftManagedClustersClient client.
func NewOpenShiftManagedClustersClientWithBaseURI(baseURI string, subscriptionID string) OpenShiftManagedClustersClient {
	return OpenShiftManagedClustersClient{NewWithBaseURI(baseURI, subscriptionID)}
}

// CreateOrUpdateAndWait creates or updates a openshift managed cluster and waits for the
// request to complete before returning.
func (client OpenShiftManagedClustersClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName, resourceName string, parameters v20180930preview.OpenShiftManagedCluster) (osmc v20180930preview.OpenShiftManagedCluster, err error) {
	var future OpenShiftManagedClustersCreateOrUpdateFuture
	future, err = client.CreateOrUpdate(ctx, resourceGroupName, resourceName, parameters)
	if err != nil {
		return
	}
	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return
	}
	return future.Result(client)
}

// CreateOrUpdate creates or updates a openshift managed cluster with the specified configuration for agents and
// OpenShift version.
// Parameters:
// resourceGroupName - the name of the resource group.
// resourceName - the name of the openshift managed cluster resource.
// parameters - parameters supplied to the Create or Update an OpenShift Managed Cluster operation.
func (client OpenShiftManagedClustersClient) CreateOrUpdate(ctx context.Context, resourceGroupName, resourceName string, parameters v20180930preview.OpenShiftManagedCluster) (result OpenShiftManagedClustersCreateOrUpdateFuture, err error) {
	if err := validation.Validate([]validation.Validation{
		{TargetValue: parameters,
			Constraints: []validation.Constraint{{Target: "parameters.Properties", Name: validation.Null, Rule: false}}}}); err != nil {
		return result, validation.NewError("containerservice.OpenShiftManagedClustersClient", "CreateOrUpdate", err.Error())
	}

	req, err := client.CreateOrUpdatePreparer(ctx, resourceGroupName, resourceName, parameters)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "CreateOrUpdate", nil, "Failure preparing request")
		return
	}

	result, err = client.CreateOrUpdateSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "CreateOrUpdate", result.Response(), "Failure sending request")
		return
	}

	return
}

// CreateOrUpdatePreparer prepares the CreateOrUpdate request.
func (client OpenShiftManagedClustersClient) CreateOrUpdatePreparer(ctx context.Context, resourceGroupName string, resourceName string, parameters v20180930preview.OpenShiftManagedCluster) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"resourceName":      autorest.Encode("path", resourceName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	queryParameters := map[string]interface{}{
		"api-version": v20180930preview.APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsPut(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/openShiftManagedClusters/{resourceName}", pathParameters),
		autorest.WithJSON(parameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// CreateOrUpdateSender sends the CreateOrUpdate request. The method will close the
// http.Response Body if it receives an error.
func (client OpenShiftManagedClustersClient) CreateOrUpdateSender(req *http.Request) (future OpenShiftManagedClustersCreateOrUpdateFuture, err error) {
	var resp *http.Response
	resp, err = autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
	if err != nil {
		return
	}
	err = autorest.Respond(resp, azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusCreated))
	if err != nil {
		return
	}
	future.Future, err = azure.NewFutureFromResponse(resp)
	return
}

// CreateOrUpdateResponder handles the response to the CreateOrUpdate request. The method always
// closes the http.Response Body.
func (client OpenShiftManagedClustersClient) CreateOrUpdateResponder(resp *http.Response) (result v20180930preview.OpenShiftManagedCluster, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusCreated),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// Delete deletes the openshift managed cluster with a specified resource group and name.
// Parameters:
// resourceGroupName - the name of the resource group.
// resourceName - the name of the openshift managed cluster resource.
func (client OpenShiftManagedClustersClient) Delete(ctx context.Context, resourceGroupName string, resourceName string) (result OpenShiftManagedClustersDeleteFuture, err error) {
	req, err := client.DeletePreparer(ctx, resourceGroupName, resourceName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "Delete", nil, "Failure preparing request")
		return
	}

	result, err = client.DeleteSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "Delete", result.Response(), "Failure sending request")
		return
	}

	return
}

// DeletePreparer prepares the Delete request.
func (client OpenShiftManagedClustersClient) DeletePreparer(ctx context.Context, resourceGroupName string, resourceName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"resourceName":      autorest.Encode("path", resourceName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	queryParameters := map[string]interface{}{
		"api-version": v20180930preview.APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsDelete(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/openShiftManagedClusters/{resourceName}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// DeleteSender sends the Delete request. The method will close the
// http.Response Body if it receives an error.
func (client OpenShiftManagedClustersClient) DeleteSender(req *http.Request) (future OpenShiftManagedClustersDeleteFuture, err error) {
	var resp *http.Response
	resp, err = autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
	if err != nil {
		return
	}
	err = autorest.Respond(resp, azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted, http.StatusNoContent))
	if err != nil {
		return
	}
	future.Future, err = azure.NewFutureFromResponse(resp)
	return
}

// DeleteResponder handles the response to the Delete request. The method always
// closes the http.Response Body.
func (client OpenShiftManagedClustersClient) DeleteResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted, http.StatusNoContent),
		autorest.ByClosing())
	result.Response = resp
	return
}

// Get gets the details of the managed openshift cluster with a specified resource group and name.
// Parameters:
// resourceGroupName - the name of the resource group.
// resourceName - the name of the openshift managed cluster resource.
func (client OpenShiftManagedClustersClient) Get(ctx context.Context, resourceGroupName string, resourceName string) (result v20180930preview.OpenShiftManagedCluster, err error) {
	req, err := client.GetPreparer(ctx, resourceGroupName, resourceName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "Get", nil, "Failure preparing request")
		return
	}

	resp, err := client.GetSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "Get", resp, "Failure sending request")
		return
	}

	result, err = client.GetResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "Get", resp, "Failure responding to request")
	}

	return
}

// GetPreparer prepares the Get request.
func (client OpenShiftManagedClustersClient) GetPreparer(ctx context.Context, resourceGroupName string, resourceName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"resourceName":      autorest.Encode("path", resourceName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	queryParameters := map[string]interface{}{
		"api-version": v20180930preview.APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/openShiftManagedClusters/{resourceName}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// GetSender sends the Get request. The method will close the
// http.Response Body if it receives an error.
func (client OpenShiftManagedClustersClient) GetSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// GetResponder handles the response to the Get request. The method always
// closes the http.Response Body.
func (client OpenShiftManagedClustersClient) GetResponder(resp *http.Response) (result v20180930preview.OpenShiftManagedCluster, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// UpdateTags updates an openshift managed cluster with the specified tags.
// Parameters:
// resourceGroupName - the name of the resource group.
// resourceName - the name of the openshift managed cluster resource.
// parameters - parameters supplied to the Update OpenShift Managed Cluster Tags operation.
func (client OpenShiftManagedClustersClient) UpdateTags(ctx context.Context, resourceGroupName string, resourceName string, parameters TagsObject) (result OpenShiftManagedClustersUpdateTagsFuture, err error) {
	req, err := client.UpdateTagsPreparer(ctx, resourceGroupName, resourceName, parameters)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "UpdateTags", nil, "Failure preparing request")
		return
	}

	result, err = client.UpdateTagsSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "UpdateTags", result.Response(), "Failure sending request")
		return
	}

	return
}

// UpdateTagsPreparer prepares the UpdateTags request.
func (client OpenShiftManagedClustersClient) UpdateTagsPreparer(ctx context.Context, resourceGroupName string, resourceName string, parameters TagsObject) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"resourceName":      autorest.Encode("path", resourceName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	queryParameters := map[string]interface{}{
		"api-version": v20180930preview.APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsPatch(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/openShiftManagedClusters/{resourceName}", pathParameters),
		autorest.WithJSON(parameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// UpdateTagsSender sends the UpdateTags request. The method will close the
// http.Response Body if it receives an error.
func (client OpenShiftManagedClustersClient) UpdateTagsSender(req *http.Request) (future OpenShiftManagedClustersUpdateTagsFuture, err error) {
	var resp *http.Response
	resp, err = autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
	if err != nil {
		return
	}
	err = autorest.Respond(resp, azure.WithErrorUnlessStatusCode(http.StatusOK))
	if err != nil {
		return
	}
	future.Future, err = azure.NewFutureFromResponse(resp)
	return
}

// UpdateTagsResponder handles the response to the UpdateTags request. The method always
// closes the http.Response Body.
func (client OpenShiftManagedClustersClient) UpdateTagsResponder(resp *http.Response) (result v20180930preview.OpenShiftManagedCluster, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// OpenShiftManagedClustersCreateOrUpdateFuture an abstraction for monitoring and retrieving the results of a
// long-running operation.
type OpenShiftManagedClustersCreateOrUpdateFuture struct {
	azure.Future
}

// Result returns the result of the asynchronous operation.
// If the operation has not completed it will return an error.
func (future *OpenShiftManagedClustersCreateOrUpdateFuture) Result(client OpenShiftManagedClustersClient) (osmc v20180930preview.OpenShiftManagedCluster, err error) {
	var done bool
	done, err = future.DoneWithContext(context.Background(), client)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersCreateOrUpdateFuture", "Result", future.Response(), "Polling failure")
		return
	}
	if !done {
		err = azure.NewAsyncOpIncompleteError("containerservice.OpenShiftManagedClustersCreateOrUpdateFuture")
		return
	}
	sender := autorest.DecorateSender(client, autorest.DoRetryForStatusCodes(client.RetryAttempts, client.RetryDuration, autorest.StatusCodesForRetry...))
	if osmc.Response.Response, err = future.GetResult(sender); err == nil && osmc.Response.Response.StatusCode != http.StatusNoContent {
		osmc, err = client.CreateOrUpdateResponder(osmc.Response.Response)
		if err != nil {
			err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersCreateOrUpdateFuture", "Result", osmc.Response.Response, "Failure responding to request")
		}
	}
	return
}

// OpenShiftManagedClustersDeleteFuture an abstraction for monitoring and retrieving the results of a long-running
// operation.
type OpenShiftManagedClustersDeleteFuture struct {
	azure.Future
}

// Result returns the result of the asynchronous operation.
// If the operation has not completed it will return an error.
func (future *OpenShiftManagedClustersDeleteFuture) Result(client OpenShiftManagedClustersClient) (ar autorest.Response, err error) {
	var done bool
	done, err = future.DoneWithContext(context.Background(), client)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersDeleteFuture", "Result", future.Response(), "Polling failure")
		return
	}
	if !done {
		err = azure.NewAsyncOpIncompleteError("containerservice.OpenShiftManagedClustersDeleteFuture")
		return
	}
	ar.Response = future.Response()
	return
}
