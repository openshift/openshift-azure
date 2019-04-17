package azureclient

import (
	"encoding/json"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/sirupsen/logrus"
)

type retryKey struct {
	clientName string
	code       string
}

var retryCodes = map[retryKey]struct{}{
	// Code="InternalExecutionError" Message="An internal execution error
	// occurred."
	{clientName: "compute.VirtualMachineScaleSetsClient", code: "InternalExecutionError"}:   {},
	{clientName: "compute.VirtualMachineScaleSetVMsClient", code: "InternalExecutionError"}: {},

	// Code="InvalidResourceReference"
	// Message="Resource
	// /.../providers/Microsoft.Network/loadBalancers/KUBERNETES-INTERNAL
	// referenced by resource
	// /.../providers/Microsoft.Compute/virtualMachineScaleSets/ss-compute-1555464513
	// was not found. Please make sure that the referenced resource exists, and
	// that both resources are in the same region."
	{clientName: "compute.VirtualMachineScaleSetVMsClient", code: "InvalidResourceReference"}: {},
}

type deploymentRetryKey struct {
	code           string
	cloudErrorCode string
}

var deploymentRetryCodes = map[deploymentRetryKey]struct{}{
	// Code="DeploymentFailed"
	// Message="At least one resource deployment operation failed. Please list
	// deployment operations for details. Please see https://aka.ms/arm-debug
	// for usage details."
	// Details=
	//   code="Conflict"
	//   message="{"error": {"code": "ResourcePurchaseCanceling",
	//     "message": "The resource 'ss-infra-1554422008' with the id
	//     'Microsoft.Compute/virtualMachineScaleSets/ss-infra-1554422008' has a
	//     previous order being canceled. Please try after some time or create
	//     resource with different name."}}"
	{code: "Conflict", cloudErrorCode: "ResourcePurchaseCanceling"}: {},

	// Code="DeploymentFailed"
	// Message="At least one resource deployment operation failed. Please list
	// deployment operations for details. Please see https://aka.ms/arm-debug
	// for usage details."
	// Details=
	//   code=InternalServerError
	//   message="{"error": {"code": "ResourceDeploymentFailure",
	//     "message": "Encountered internal server error. Diagnostic
	//     information: timestamp '20190412T193259Z', subscription id '...',
	//     tracking id '...', request correlation id '...'."}}"
	{code: "InternalServerError", cloudErrorCode: "ResourceDeploymentFailure"}: {},
}

type retrySender struct {
	autorest.Sender
	log        *logrus.Entry
	clientName string
}

func (rs *retrySender) Do(req *http.Request) (resp *http.Response, err error) {
	retry, retries := 0, 3
	for {
		retry++

		resp, err = rs.Sender.Do(req)
		if err == nil {
			return
		}

		if retry <= retries && isRetryableError(rs.clientName, err) {
			rs.log.Warnf("%s: retry %d", err, retry)
			continue
		}

		return
	}
}

func isRetryableError(clientName string, err error) bool {
	re, ok := err.(*azure.RequestError)
	if !ok || re.ServiceError == nil {
		return false
	}

	if _, found := retryCodes[retryKey{clientName: clientName, code: re.ServiceError.Code}]; found {
		return true
	}

	if re.ServiceError.Code == "DeploymentFailed" {
		for _, detail := range re.ServiceError.Details {
			code, ok := detail["code"].(string)
			if !ok {
				continue
			}

			message, ok := detail["message"].(string)
			if !ok {
				continue
			}

			var ce compute.CloudError
			if json.Unmarshal([]byte(message), &ce) == nil {
				if ce.Error != nil && ce.Error.Code != nil {
					if _, found := deploymentRetryCodes[deploymentRetryKey{code: code, cloudErrorCode: *ce.Error.Code}]; found {
						return true
					}
				}
			}
		}
	}

	return false
}
