package validate

import (
	"net"
	"regexp"

	"github.com/openshift/openshift-azure/pkg/api"
)

var (
	RxRfc1123 = regexp.MustCompile(`(?i)^` +
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

	// This regexp is to check image version format
	imageVersion = regexp.MustCompile(`^[0-9]{3}.[0-9]{1,4}.[0-9]{8}$`)

	// This regexp is to check cluster version (plugin) format
	clusterVersion = regexp.MustCompile(`^v\d+\.\d+$`)

	validMasterAndInfraVMSizes = map[api.VMSize]struct{}{
		// Rationale here is: a highly limited set of modern general purpose
		// offerings which we can reason about and test for now.

		// GENERAL PURPOSE VMS

		api.StandardD4sV3:  {},
		api.StandardD8sV3:  {},
		api.StandardD16sV3: {},
		api.StandardD32sV3: {},
		api.StandardD64sV3: {},

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
		api.StandardD64sV3: {},

		api.StandardDS4V2: {},
		api.StandardDS5V2: {},

		// COMPUTE OPTIMIZED VMS

		api.StandardF8sV2:  {},
		api.StandardF16sV2: {},
		api.StandardF32sV2: {},
		api.StandardF64sV2: {},
		api.StandardF72sV2: {},

		api.StandardF8s:  {},
		api.StandardF16s: {},

		// MEMORY OPTIMIZED VMS

		api.StandardE4sV3:  {},
		api.StandardE8sV3:  {},
		api.StandardE16sV3: {},
		api.StandardE20sV3: {},
		api.StandardE32sV3: {},
		api.StandardE64sV3: {},

		// Skipping StandardM* for now as they may require significant extra
		// effort/spend to certify and support.

		api.StandardGS2: {},
		api.StandardGS3: {},
		api.StandardGS4: {},
		api.StandardGS5: {},

		api.StandardDS12V2: {},
		api.StandardDS13V2: {},
		api.StandardDS14V2: {},
		api.StandardDS15V2: {},

		// STORAGE OPTIMIZED VMS

		api.StandardL4s:  {},
		api.StandardL8s:  {},
		api.StandardL16s: {},
		api.StandardL32s: {},
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
