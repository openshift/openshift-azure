package validate

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	uuid "github.com/satori/go.uuid"

	"github.com/openshift/openshift-azure/pkg/api"
)

var (
	rxClusterName = regexp.MustCompile(`^[-\w\._\(\)]+$`)

	rxLocation = regexp.MustCompile(`(?i)^[a-z0-9]+$`)

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

	rxWorkspaceResourceID = regexp.MustCompile(`(?i)^` +
		`/subscriptions/[^/]+` +
		`/resourceGroups/[^/]+` +
		`/providers/Microsoft\.OperationalInsights` +
		`/workspaces/[^/]+` +
		`$`)

	rxAgentPoolProfileName = regexp.MustCompile(`^[a-z0-9]{1,12}$`)

	rxRouterProfileName = rxAgentPoolProfileName

	// This regexp is to guard against InvalidDomainNameLabel for hostname validation
	rxCloudDomainLabel = regexp.MustCompile(`^[a-z][a-z0-9-]{1,61}[a-z0-9]\.`)

	// This regexp is to check image version format
	rxImageVersion = regexp.MustCompile(`^[0-9]{3}.[0-9]{1,4}.[0-9]{8}$`)

	// This regexp is to check plugin version format
	rxPluginVersion = regexp.MustCompile(`^v\d+\.\d+$`)

	// This regexp checks rpm package name format
	rxRpmPackage = regexp.MustCompile(`^[a-zA-Z0-9_\-\.+]+$`)

	rxResourceGroupName = regexp.MustCompile(`^[-\w._()]{1,90}$`)

	rxKeyVaultSecretURL = regexp.MustCompile(`^https://[-a-zA-Z0-9]{3,24}\.vault\.azure\.net/secrets/[-a-zA-Z0-9]{1,127}$`)

	validMasterAndInfraVMSizes = map[api.VMSize]struct{}{
		// Rationale here is: a highly limited set of modern general purpose
		// offerings which we can reason about and test for now.

		// General purpose VMs
		api.StandardD4sV3:  {},
		api.StandardD8sV3:  {},
		api.StandardD16sV3: {},
		api.StandardD32sV3: {},

		// TODO: enable vertical scaling of existing OSA clusters.
	}

	validComputeVMSizes = map[api.VMSize]struct{}{
		// Rationale here is: modern offerings per (general purpose /
		// memory-optimized / compute-optimized) family, with at least 16GiB
		// RAM, 32GiB SSD, 8 data disks, premium storage support.  GPU and HPC
		// use cases are probably blocked for now on drivers / multiple agent
		// pool support.

		// General purpose VMs
		api.StandardD4sV3:  {},
		api.StandardD8sV3:  {},
		api.StandardD16sV3: {},
		api.StandardD32sV3: {},

		// Memory optimized VMs
		api.StandardE4sV3:  {},
		api.StandardE8sV3:  {},
		api.StandardE16sV3: {},
		api.StandardE32sV3: {},

		// Compute optimized VMs
		api.StandardF8sV2:  {},
		api.StandardF16sV2: {},
		api.StandardF32sV2: {},
	}

	clusterNetworkCIDR *net.IPNet
	serviceNetworkCIDR *net.IPNet

	validAgentPoolProfileRoles = map[api.AgentPoolProfileRole]struct{}{
		api.AgentPoolProfileRoleCompute: {},
		api.AgentPoolProfileRoleInfra:   {},
		api.AgentPoolProfileRoleMaster:  {},
	}

	validRouterProfileNames = map[string]struct{}{
		"default": {},
	}
)

func init() {
	var err error

	// TODO: we probably need to bite the bullet and make these configurable.
	_, clusterNetworkCIDR, err = net.ParseCIDR("10.128.0.0/14")
	if err != nil {
		panic(err)
	}

	_, serviceNetworkCIDR, err = net.ParseCIDR("172.30.0.0/16")
	if err != nil {
		panic(err)
	}
}

func isValidClusterName(n string) bool {
	return rxClusterName.MatchString(n)
}

func isValidLocation(l string) bool {
	return rxLocation.MatchString(l)
}

func isValidHostname(h string) bool {
	return len(h) <= 255 && rxRfc1123.MatchString(h)
}

func isValidCloudAppHostname(h, location string) bool {
	if !rxCloudDomainLabel.MatchString(h) {
		return false
	}
	return strings.HasSuffix(h, "."+location+".cloudapp.azure.com") && strings.Count(h, ".") == 4
}

func isValidIPV4CIDR(cidr string) bool {
	ip, net, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	if ip.To4() == nil {
		return false
	}
	if net == nil || !ip.Equal(net.IP) {
		return false
	}
	return true
}

func IsValidBlobName(c string) bool {
	// https://docs.microsoft.com/en-us/rest/api/storageservices/naming-and-referencing-containers--blobs--and-metadata
	if len(c) < 1 || len(c) > 1024 {
		return false
	}
	if strings.HasSuffix(c, ".") || strings.HasSuffix(c, "/") {
		return false
	}
	if strings.Contains(c, "./") || strings.Contains(c, "/.") {
		return false
	}
	u, err := url.Parse(fmt.Sprintf("http://example.com/%s", c))
	if err != nil {
		return false
	}
	if u.EscapedPath() != "/"+c {
		return false
	}
	return true
}

func vnetContainsSubnet(vnet, subnet *net.IPNet) bool {
	vnetbits, _ := vnet.Mask.Size()
	subnetbits, _ := subnet.Mask.Size()
	if vnetbits > subnetbits {
		// e.g., vnet is a /24, subnet is a /16: vnet cannot contain subnet.
		return false
	}

	return vnet.IP.Equal(subnet.IP.Mask(vnet.Mask))
}

func isValidUUID(u string) bool {
	_, err := uuid.FromString(u)
	return err == nil
}

func IsValidAgentPoolHostname(hostname string) bool {
	parts := strings.Split(hostname, "-")
	switch len(parts) {
	case 2: // master-XXXXXX
		if parts[0] != "master" ||
			len(parts[1]) != 6 {
			return false
		}
		_, err := strconv.ParseUint(parts[1], 36, 64)
		if err != nil {
			return false
		}

	case 3: // something-XXXXXXXXXX-XXXXXX
		if !rxAgentPoolProfileName.MatchString(parts[0]) ||
			parts[0] == "master" ||
			len(parts[1]) != 10 ||
			len(parts[2]) != 6 {
			return false
		}
		_, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return false
		}
		_, err = strconv.ParseUint(parts[2], 36, 64)
		if err != nil {
			return false
		}

	default:
		return false
	}

	return true
}

func isValidMasterAndInfraVMSize(size api.VMSize, runningUnderTest bool) bool {
	if runningUnderTest && size == api.StandardD2sV3 {
		return true
	}

	_, found := validMasterAndInfraVMSizes[size]
	return found
}

func isValidComputeVMSize(size api.VMSize, runningUnderTest bool) bool {
	if runningUnderTest && size == api.StandardD2sV3 {
		return true
	}

	_, found := validComputeVMSizes[size]
	return found
}

func isValidRpmPackageName(name string) bool {
	if strings.HasSuffix(name, ".rpm") || !rxRpmPackage.MatchString(name) {
		return false
	}
	return true
}
