package validate

import (
	"net"
	"regexp"

	"github.com/openshift/openshift-azure/pkg/api"
)

var (
	rxRfc1123 = regexp.MustCompile(`(?i)^` +
		`([a-z0-9]|[a-z0-9][-a-z0-9]{0,61}[a-z0-9])` +
		`(\.([a-z0-9]|[a-z0-9][-a-z0-9]{0,61}[a-z0-9]))*` +
		`$`)

	rxVNetID = regexp.MustCompile(`(?i)^` +
		`/subscriptions/[^/]+` +
		`/resourceGroups/[^/]+` +
		`/providers/Microsoft\.Network` +
		`/virtualNetworks/[^/]+` +
		`$`)

	rxAgentPoolProfileName = regexp.MustCompile(`^[a-z0-9]{1,12}$`)

	// This regexp is to guard against InvalidDomainNameLabel for hostname validation
	rxCloudDomainLabel = regexp.MustCompile(`^[a-z][a-z0-9-]{1,61}[a-z0-9]\.`)

	rxBlobContainerName = regexp.MustCompile(`^[a-z0-9-]{3,63}$`)

	// This regexp is to check image version format
	imageVersion = regexp.MustCompile(`^[0-9]{3}.[0-9]{1,4}.[0-9]{8}$`)

	// This regexp is to check plgin version format
	pluginVersion = regexp.MustCompile(`^v\d+\.\d+$`)

	validMasterAndInfraVMSizes = map[api.VMSize]struct{}{
		// Rationale here is: a highly limited set of modern general purpose
		// offerings which we can reason about and test for now.

		// GENERAL PURPOSE VMS

		api.StandardD4sV3:  {},
		api.StandardD8sV3:  {},
		api.StandardD16sV3: {},
		api.StandardD32sV3: {},

		// TODO: consider enabling storage optimized (Ls) series for masters and
		// moving the etcd onto the VM SSD storage.

		// TODO: enable vertical scaling of existing OSA clusters.
	}

	validComputeVMSizes = map[api.VMSize]struct{}{
		// Rationale here is: modern offerings per (general purpose /
		// memory-optimized / compute-optimized / storage-optimized) family,
		// with at least 16GiB RAM, 32GiB SSD, 8 data disks, premium storage
		// support.  GPU and HPC use cases are probably blocked for now on
		// drivers / multiple agent pool support.

		// GENERAL PURPOSE VMS

		// Skiping StandardB* (burstable) VMs for now as they could be hard to
		// reason about performance-wise.

		api.StandardD4sV3:  {},
		api.StandardD8sV3:  {},
		api.StandardD16sV3: {},
		api.StandardD32sV3: {},

		// COMPUTE OPTIMIZED VMS

		api.StandardF8sV2:  {},
		api.StandardF16sV2: {},
		api.StandardF32sV2: {},
	}
)

var (
	clusterNetworkCIDR *net.IPNet
	serviceNetworkCIDR *net.IPNet
)

// APIValidator validator for external API
type APIValidator struct {
	runningUnderTest bool
}

// AdminAPIValidator validator for external Admin API
type AdminAPIValidator struct {
	runningUnderTest bool
}

// PluginAPIValidator validator for external Plugin API
type PluginAPIValidator struct{}

// NewAPIValidator return instance of external API validator
func NewAPIValidator(runningUnderTest bool) *APIValidator {
	return &APIValidator{runningUnderTest: runningUnderTest}
}

// NewAdminValidator return instance of external Admin API validator
func NewAdminValidator(runningUnderTest bool) *AdminAPIValidator {
	return &AdminAPIValidator{runningUnderTest: runningUnderTest}
}

// NewPluginAPIValidator return instance of external Plugin API validator
func NewPluginAPIValidator() *PluginAPIValidator {
	return &PluginAPIValidator{}
}
