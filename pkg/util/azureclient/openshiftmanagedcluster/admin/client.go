package api

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"

	"github.com/openshift/openshift-azure/pkg/api/admin/api"
)

// BaseClient is the base client for Containerservice.
type BaseClient struct {
	autorest.Client
	BaseURI        string
	SubscriptionID string
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
	return "openshift-azure/pkg/api/admin/api/client"
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
func (future *OpenShiftManagedClustersUpdateTagsFuture) Result(client OpenShiftManagedClustersClient) (osmc api.OpenShiftManagedCluster, err error) {
	var done bool
	done, err = future.Done(client)
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

// NewOpenShiftManagedClustersClientWithBaseURI creates an instance of the OpenShiftManagedClustersClient client.
func NewOpenShiftManagedClustersClientWithBaseURI(baseURI string, subscriptionID string) OpenShiftManagedClustersClient {
	return OpenShiftManagedClustersClient{NewWithBaseURI(baseURI, subscriptionID)}
}

// OpenShiftManagedClustersBackupFuture is an abstraction for monitoring and retrieving the results of a
// long-running operation.
type OpenShiftManagedClustersBackupFuture struct {
	azure.Future
}

// Result returns the result of the asynchronous operation.
// If the operation has not completed it will return an error.
func (future *OpenShiftManagedClustersBackupFuture) Result(client OpenShiftManagedClustersClient) (ar autorest.Response, err error) {
	var done bool
	done, err = future.Done(client)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersBackupFuture", "Result", future.Response(), "Polling failure")
		return
	}
	if !done {
		err = azure.NewAsyncOpIncompleteError("containerservice.OpenShiftManagedClustersBackupFuture")
		return
	}
	ar.Response = future.Response()
	return
}

// BackupAndWait backs up an  OpenShiftManagedCluster
// and waits for the request to complete before returning.
func (client OpenShiftManagedClustersClient) BackupAndWait(ctx context.Context, resourceGroupName, resourceName, backupName string) (result autorest.Response, err error) {
	var future OpenShiftManagedClustersBackupFuture
	future, err = client.Backup(ctx, resourceGroupName, resourceName, backupName)
	if err != nil {
		return
	}
	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return
	}
	return future.Result(client)
}

// Backup backs up an OpenShiftManagedCluster
// Parameters:
// resourceGroupName - the name of the Resource group.
// resourceName - the name of the OpenShiftManagedCluster
func (client OpenShiftManagedClustersClient) Backup(ctx context.Context, resourceGroupName, resourceName, backupName string) (result OpenShiftManagedClustersBackupFuture, err error) {
	req, err := client.BackupPreparer(ctx, resourceGroupName, resourceName, backupName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "Backup", nil, "Failure preparing request")
		return
	}

	result, err = client.BackupSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "Backup", result.Response(), "Failure sending request")
		return
	}

	return
}

// BackupPreparer prepares the Backup request.
func (client OpenShiftManagedClustersClient) BackupPreparer(ctx context.Context, resourceGroupName, resourceName, backupName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"resourceName":      autorest.Encode("path", resourceName),
		"backupName":        autorest.Encode("path", backupName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	queryParameters := map[string]interface{}{
		"api-version": api.APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsPut(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/openShiftManagedClusters/{resourceName}/backup/{backupName}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// BackupSender sends the Backup request. The method will close the
// http.Response Body if it receives an error.
func (client OpenShiftManagedClustersClient) BackupSender(req *http.Request) (future OpenShiftManagedClustersBackupFuture, err error) {
	var resp *http.Response
	resp, err = autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
	if err != nil {
		return
	}
	future.Future, err = azure.NewFutureFromResponse(resp)
	return
}

// BackupResponder handles the response to the Backup request. The method
// always closes the http.Response Body.
func (client OpenShiftManagedClustersClient) BackupResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted),
		autorest.ByClosing())
	result.Response = resp
	return
}

// CreateOrUpdateAndWait creates or updates a openshift managed cluster and waits for the
// request to complete before returning.
func (client OpenShiftManagedClustersClient) CreateOrUpdateAndWait(ctx context.Context, resourceGroupName, resourceName string, parameters api.OpenShiftManagedCluster) (osmc api.OpenShiftManagedCluster, err error) {
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
func (client OpenShiftManagedClustersClient) CreateOrUpdate(ctx context.Context, resourceGroupName, resourceName string, parameters api.OpenShiftManagedCluster) (result OpenShiftManagedClustersCreateOrUpdateFuture, err error) {
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
func (client OpenShiftManagedClustersClient) CreateOrUpdatePreparer(ctx context.Context, resourceGroupName string, resourceName string, parameters api.OpenShiftManagedCluster) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"resourceName":      autorest.Encode("path", resourceName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	queryParameters := map[string]interface{}{
		"api-version": api.APIVersion,
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
func (client OpenShiftManagedClustersClient) CreateOrUpdateResponder(resp *http.Response) (result api.OpenShiftManagedCluster, err error) {
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
		"api-version": api.APIVersion,
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

// OpenShiftManagedClustersForceUpdateFuture is an abstraction for monitoring and retrieving the results of a
// long-running operation.
type OpenShiftManagedClustersForceUpdateFuture struct {
	azure.Future
}

// Result returns the result of the asynchronous operation.
// If the operation has not completed it will return an error.
func (future *OpenShiftManagedClustersForceUpdateFuture) Result(client OpenShiftManagedClustersClient) (ar autorest.Response, err error) {
	var done bool
	done, err = future.Done(client)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersForceUpdateFuture", "Result", future.Response(), "Polling failure")
		return
	}
	if !done {
		err = azure.NewAsyncOpIncompleteError("containerservice.OpenShiftManagedClustersForceUpdateFuture")
		return
	}
	ar.Response = future.Response()
	return
}

// ForceUpdateAndWait zeroes the update hash to force an update to an OpenShiftManagedCluster
// and waits for the request to complete before returning.
func (client OpenShiftManagedClustersClient) ForceUpdateAndWait(ctx context.Context, resourceGroupName, resourceName string) (result autorest.Response, err error) {
	var future OpenShiftManagedClustersForceUpdateFuture
	future, err = client.ForceUpdate(ctx, resourceGroupName, resourceName)
	if err != nil {
		return
	}
	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return
	}
	return future.Result(client)
}

// ForceUpdate zeroes the update hash to force an update to an OpenShiftManagedCluster
// Parameters:
// resourceGroupName - the name of the Resource group.
// resourceName - the name of the OpenShiftManagedCluster
func (client OpenShiftManagedClustersClient) ForceUpdate(ctx context.Context, resourceGroupName, resourceName string) (result OpenShiftManagedClustersForceUpdateFuture, err error) {
	req, err := client.ForceUpdatePreparer(ctx, resourceGroupName, resourceName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "ForceUpdate", nil, "Failure preparing request")
		return
	}

	result, err = client.ForceUpdateSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "ForceUpdate", result.Response(), "Failure sending request")
		return
	}

	return
}

// ForceUpdatePreparer prepares the ForceUpdate request.
func (client OpenShiftManagedClustersClient) ForceUpdatePreparer(ctx context.Context, resourceGroupName, resourceName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"resourceName":      autorest.Encode("path", resourceName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	queryParameters := map[string]interface{}{
		"api-version": api.APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsPut(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/openShiftManagedClusters/{resourceName}/forceUpdate", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// ForceUpdateSender sends the ForceUpdate request. The method will close the
// http.Response Body if it receives an error.
func (client OpenShiftManagedClustersClient) ForceUpdateSender(req *http.Request) (future OpenShiftManagedClustersForceUpdateFuture, err error) {
	var resp *http.Response
	resp, err = autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
	if err != nil {
		return
	}
	future.Future, err = azure.NewFutureFromResponse(resp)
	return
}

// ForceUpdateResponder handles the response to the ForceUpdate request. The method
// always closes the http.Response Body.
func (client OpenShiftManagedClustersClient) ForceUpdateResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted),
		autorest.ByClosing())
	result.Response = resp
	return
}

// Get gets the details of the managed openshift cluster with a specified resource group and name.
// Parameters:
// resourceGroupName - the name of the resource group.
// resourceName - the name of the openshift managed cluster resource.
func (client OpenShiftManagedClustersClient) Get(ctx context.Context, resourceGroupName string, resourceName string) (result api.OpenShiftManagedCluster, err error) {
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
		"api-version": api.APIVersion,
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
func (client OpenShiftManagedClustersClient) GetResponder(resp *http.Response) (result api.OpenShiftManagedCluster, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// GetControlPlanePods gets the details of the managed openshift cluster with a specified resource group and name.
// Parameters:
// resourceGroupName - the name of the resource group.
// resourceName - the name of the openshift managed cluster resource.
func (client OpenShiftManagedClustersClient) GetControlPlanePods(ctx context.Context, resourceGroupName string, resourceName string) (result OpenShiftManagedClustersControlPlanePods, err error) {
	req, err := client.GetControlPlanePodsPreparer(ctx, resourceGroupName, resourceName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "GetControlPlanePods", nil, "Failure preparing request")
		return
	}

	resp, err := client.GetControlPlanePodsSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "GetControlPlanePods", resp, "Failure sending request")
		return
	}

	result, err = client.GetControlPlanePodsResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "GetControlPlanePods", resp, "Failure responding to request")
	}

	return
}

// GetControlPlanePodsPreparer prepares the GetControlPlanePods request.
func (client OpenShiftManagedClustersClient) GetControlPlanePodsPreparer(ctx context.Context, resourceGroupName string, resourceName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"resourceName":      autorest.Encode("path", resourceName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	queryParameters := map[string]interface{}{
		"api-version": api.APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/openShiftManagedClusters/{resourceName}/status", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// GetControlPlanePodsSender sends the GetControlPlanePods request. The method will close the
// http.Response Body if it receives an error.
func (client OpenShiftManagedClustersClient) GetControlPlanePodsSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// GetControlPlanePodsResponder handles the response to the GetControlPlanePods request. The method always
// closes the http.Response Body.
func (client OpenShiftManagedClustersClient) GetControlPlanePodsResponder(resp *http.Response) (result OpenShiftManagedClustersControlPlanePods, err error) {
	defer resp.Body.Close()
	status, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	result.Response = autorest.Response{Response: resp}
	result.Items = status
	return
}

// ListClusterVMs gets the details of the managed openshift cluster with a specified resource group and name.
// Parameters:
// resourceGroupName - the name of the resource group.
// resourceName - the name of the openshift managed cluster resource.
func (client OpenShiftManagedClustersClient) ListClusterVMs(ctx context.Context, resourceGroupName string, resourceName string) (result OpenShiftManagedClustersClusterVMs, err error) {
	req, err := client.ListClusterVMsPreparer(ctx, resourceGroupName, resourceName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "ListClusterVMs", nil, "Failure preparing request")
		return
	}

	resp, err := client.ListClusterVMsSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "ListClusterVMs", resp, "Failure sending request")
		return
	}

	result, err = client.ListClusterVMsResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "ListClusterVMs", resp, "Failure responding to request")
	}

	return
}

// ListClusterVMsPreparer prepares the ListClusterVMs request.
func (client OpenShiftManagedClustersClient) ListClusterVMsPreparer(ctx context.Context, resourceGroupName string, resourceName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"resourceName":      autorest.Encode("path", resourceName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	queryParameters := map[string]interface{}{
		"api-version": api.APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/openShiftManagedClusters/{resourceName}/listClusterVMs", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// ListClusterVMsSender sends the ListClusterVMs request. The method will close the
// http.Response Body if it receives an error.
func (client OpenShiftManagedClustersClient) ListClusterVMsSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// ListClusterVMsResponder handles the response to the ListClusterVMs request. The method always
// closes the http.Response Body.
func (client OpenShiftManagedClustersClient) ListClusterVMsResponder(resp *http.Response) (result OpenShiftManagedClustersClusterVMs, err error) {
	defer resp.Body.Close()
	clusterVMs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	result.Response = autorest.Response{Response: resp}
	err = json.Unmarshal(clusterVMs, &result.VMs)
	return
}

// OpenShiftManagedClustersReimageFuture is an abstraction for monitoring and retrieving the results of a
// long-running operation.
type OpenShiftManagedClustersReimageFuture struct {
	azure.Future
}

// Result returns the result of the asynchronous operation.
// If the operation has not completed it will return an error.
func (future *OpenShiftManagedClustersReimageFuture) Result(client OpenShiftManagedClustersClient) (ar autorest.Response, err error) {
	var done bool
	done, err = future.Done(client)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersReimageFuture", "Result", future.Response(), "Polling failure")
		return
	}
	if !done {
		err = azure.NewAsyncOpIncompleteError("containerservice.OpenShiftManagedClustersReimageFuture")
		return
	}
	ar.Response = future.Response()
	return
}

// OpenShiftManagedClustersVMReimageParameters describes the parameters for reimaging
// a VM in a scale set in an OpenShiftManagedCluster
type OpenShiftManagedClustersVMReimageParameters struct {
	// TempDisk - Specifies whether to reimage temp disk. Default value: false.
	TempDisk *bool `json:"tempDisk,omitempty"`
}

// ReimageAndWait reimages a VirtualMachine within a VirtualMachineScaleSet in an
// OpenshiftManagedCluster and waits for the request to complete before returning.
func (client OpenShiftManagedClustersClient) ReimageAndWait(ctx context.Context, resourceGroupName, resourceName, hostname string) (result autorest.Response, err error) {
	var future OpenShiftManagedClustersReimageFuture
	future, err = client.Reimage(ctx, resourceGroupName, resourceName, hostname)
	if err != nil {
		return
	}
	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return
	}
	return future.Result(client)
}

// Reimage reimages a VirtualMachine in an OpenshiftManagedCluster with the following parameters
// Parameters:
// resourceGroupName - the name of the Resource group
// resourceName - the name of the openshift managed cluster resource
// hostname - the hostname of a virtual machine in the cluster
func (client OpenShiftManagedClustersClient) Reimage(ctx context.Context, resourceGroupName, resourceName, hostname string) (result OpenShiftManagedClustersReimageFuture, err error) {
	req, err := client.ReimagePreparer(ctx, resourceGroupName, resourceName, hostname)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "Reimage", nil, "Failure preparing request")
		return
	}

	result, err = client.ReimageSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "Reimage", result.Response(), "Failure sending request")
		return
	}

	return
}

// ReimagePreparer prepares the Reimage request.
func (client OpenShiftManagedClustersClient) ReimagePreparer(ctx context.Context, resourceGroupName, resourceName, hostname string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"resourceName":      autorest.Encode("path", resourceName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
		"hostname":          autorest.Encode("path", hostname),
	}

	queryParameters := map[string]interface{}{
		"api-version": api.APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsPut(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/openShiftManagedClusters/{resourceName}/reimage/{hostname}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// ReimageSender sends the Reimage request. The method will close the
// http.Response Body if it receives an error.
func (client OpenShiftManagedClustersClient) ReimageSender(req *http.Request) (future OpenShiftManagedClustersReimageFuture, err error) {
	var resp *http.Response
	resp, err = autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
	if err != nil {
		return
	}
	future.Future, err = azure.NewFutureFromResponse(resp)
	return
}

// ReimageResponder handles the response to the Reimage request. The method
// always closes the http.Response Body.
func (client OpenShiftManagedClustersClient) ReimageResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted),
		autorest.ByClosing())
	result.Response = resp
	return
}

// RestoreAndWait restores an openshift managed cluster and waits for the
// request to complete before returning.
func (client OpenShiftManagedClustersClient) RestoreAndWait(ctx context.Context, resourceGroupName, resourceName string, blobName string) (result autorest.Response, err error) {
	var future OpenShiftManagedClustersRestoreFuture
	future, err = client.Restore(ctx, resourceGroupName, resourceName, blobName)
	if err != nil {
		return
	}
	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return
	}
	return future.Result(client)
}

// Restore restores an openshift managed cluster with the specified configuration for agents and
// OpenShift version.
// Parameters:
// resourceGroupName - the name of the resource group.
// resourceName - the name of the openshift managed cluster resource.
// blobName - the name of the blob from where to restore the cluster.
func (client OpenShiftManagedClustersClient) Restore(ctx context.Context, resourceGroupName, resourceName, blobName string) (result OpenShiftManagedClustersRestoreFuture, err error) {
	if blobName == "" {
		return result, validation.NewError("containerservice.OpenShiftManagedClustersClient", "Restore", "blob name cannot be empty")
	}

	req, err := client.RestorePreparer(ctx, resourceGroupName, resourceName, blobName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "Restore", nil, "Failure preparing request")
		return
	}

	result, err = client.RestoreSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "Restore", result.Response(), "Failure sending request")
		return
	}

	return
}

// RestorePreparer prepares the Restore request.
func (client OpenShiftManagedClustersClient) RestorePreparer(ctx context.Context, resourceGroupName string, resourceName string, blobName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"resourceName":      autorest.Encode("path", resourceName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	queryParameters := map[string]interface{}{
		"api-version": api.APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsPut(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/openShiftManagedClusters/{resourceName}/restore", pathParameters),
		autorest.WithJSON(blobName),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// RestoreSender sends the Restore request. The method will close the
// http.Response Body if it receives an error.
func (client OpenShiftManagedClustersClient) RestoreSender(req *http.Request) (future OpenShiftManagedClustersRestoreFuture, err error) {
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

// RestoreResponder handles the response to the Restore request. The method always
// closes the http.Response Body.
func (client OpenShiftManagedClustersClient) RestoreResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted),
		autorest.ByClosing())
	result.Response = resp
	return
}

// RotateSecretsAndWait rotates the keys of an openshift managed cluster and waits
// for the request to complete before returning.
func (client OpenShiftManagedClustersClient) RotateSecretsAndWait(ctx context.Context, resourceGroupName, resourceName string) (result autorest.Response, err error) {
	var future OpenShiftManagedClustersRotateSecretsFuture
	future, err = client.RotateSecrets(ctx, resourceGroupName, resourceName)
	if err != nil {
		return
	}
	if err = future.WaitForCompletionRef(ctx, client.Client); err != nil {
		return
	}
	return future.Result(client)
}

// RotateSecrets rotates the secrets of an openshift managed cluster with the specified
// configuration for agents and OpenShift version.
// Parameters:
// resourceGroupName - the name of the resource group.
// resourceName - the name of the openshift managed cluster resource.
func (client OpenShiftManagedClustersClient) RotateSecrets(ctx context.Context, resourceGroupName, resourceName string) (result OpenShiftManagedClustersRotateSecretsFuture, err error) {
	req, err := client.RotateSecretsPreparer(ctx, resourceGroupName, resourceName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "RotateSecrets", nil, "Failure preparing request")
		return
	}

	result, err = client.RotateSecretsSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersClient", "RotateSecrets", result.Response(), "Failure sending request")
		return
	}

	return
}

// RotateSecretsPreparer prepares the secrets rotation request.
func (client OpenShiftManagedClustersClient) RotateSecretsPreparer(ctx context.Context, resourceGroupName string, resourceName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"resourceName":      autorest.Encode("path", resourceName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	queryParameters := map[string]interface{}{
		"api-version": api.APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsPut(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/openShiftManagedClusters/{resourceName}/rotate/secrets", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// RotateSecretsSender sends the secrets rotation request. The method will close the
// http.Response Body if it receives an error.
func (client OpenShiftManagedClustersClient) RotateSecretsSender(req *http.Request) (future OpenShiftManagedClustersRotateSecretsFuture, err error) {
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

// RotateSecretsResponder handles the response to the secrets rotation request. The method
// always closes the http.Response Body.
func (client OpenShiftManagedClustersClient) RotateSecretsResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted),
		autorest.ByClosing())
	result.Response = resp
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
		"api-version": api.APIVersion,
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
func (client OpenShiftManagedClustersClient) UpdateTagsResponder(resp *http.Response) (result api.OpenShiftManagedCluster, err error) {
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
func (future *OpenShiftManagedClustersCreateOrUpdateFuture) Result(client OpenShiftManagedClustersClient) (osmc api.OpenShiftManagedCluster, err error) {
	var done bool
	done, err = future.Done(client)
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
	done, err = future.Done(client)
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

// OpenShiftManagedClustersRestoreFuture an abstraction for monitoring and retrieving the results of a
// long-running operation.
type OpenShiftManagedClustersRestoreFuture struct {
	azure.Future
}

// Result returns the result of the asynchronous operation.
// If the operation has not completed it will return an error.
func (future *OpenShiftManagedClustersRestoreFuture) Result(client OpenShiftManagedClustersClient) (ar autorest.Response, err error) {
	var done bool
	done, err = future.Done(client)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersRestoreFuture", "Result", future.Response(), "Polling failure")
		return
	}
	if !done {
		err = azure.NewAsyncOpIncompleteError("containerservice.OpenShiftManagedClustersRestoreFuture")
		return
	}
	ar.Response = future.Response()
	return
}

// OpenShiftManagedClustersRotateSecretsFuture an abstraction for monitoring and retrieving the results of a
// long-running key rotation operation.
type OpenShiftManagedClustersRotateSecretsFuture struct {
	azure.Future
}

// Result returns the result of the asynchronous operation.
// If the operation has not completed it will return an error.
func (future *OpenShiftManagedClustersRotateSecretsFuture) Result(client OpenShiftManagedClustersClient) (ar autorest.Response, err error) {
	var done bool
	done, err = future.Done(client)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersKeyRotationFuture", "Result", future.Response(), "Polling failure")
		return
	}
	if !done {
		err = azure.NewAsyncOpIncompleteError("containerservice.OpenShiftManagedClustersKeyRotationFuture")
		return
	}
	ar.Response = future.Response()
	return
}

// OpenShiftManagedClustersGetControlPlanePodsFuture an abstraction for monitoring and retrieving the results of a
// cluster status operation.
type OpenShiftManagedClustersGetControlPlanePodsFuture struct {
	azure.Future
}

func (future *OpenShiftManagedClustersGetControlPlanePodsFuture) Result(client OpenShiftManagedClustersClient) (ar autorest.Response, err error) {
	var done bool
	done, err = future.Done(client)
	if err != nil {
		err = autorest.NewErrorWithError(err, "containerservice.OpenShiftManagedClustersGetControlPlanePodsFuture", "Result", future.Response(), "Polling failure")
		return
	}
	if !done {
		err = azure.NewAsyncOpIncompleteError("containerservice.OpenShiftManagedClustersGetControlPlanePodsFuture")
		return
	}
	ar.Response = future.Response()
	return
}

// OpenShiftManagedClustersControlPlanePods contains the status of the control plane pods
type OpenShiftManagedClustersControlPlanePods struct {
	autorest.Response `json:"-"`

	Items []byte `json:"-"`
}

// OpenShiftManagedClustersClusterVMs contains the status of the cluster VMs
type OpenShiftManagedClustersClusterVMs struct {
	autorest.Response `json:"-"`

	VMs []string `json:"-"`
}
