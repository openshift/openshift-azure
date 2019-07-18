package validate

import (
	"fmt"
	"net"
	"testing"
	"time"
)

func TestValidateImageVersion(t *testing.T) {
	invalidVersions := []string{
		".1.1",
		"1.1",
		"1.1.123456789",
		"1.12345.1",
		"1234.1.1",
		"1.1",
		"1",
	}
	for _, invalidVersion := range invalidVersions {
		if rxImageVersion.MatchString(invalidVersion) {
			t.Errorf("invalid Version passed test: %s", invalidVersion)
		}
	}
	validVersions := []string{
		"123.1.12345678",
		"123.123.12345678",
		"123.1234.12345678",
	}
	for _, validVersion := range validVersions {
		if !rxImageVersion.MatchString(validVersion) {
			t.Errorf("valid Version failed to test: %s", validVersion)
		}
	}
}

func TestValidatePluginVersion(t *testing.T) {
	invalidVersions := []string{
		".1.1",
		"1.1",
		"1.1.1",
		"v1",
		"v1.0.0",
	}
	for _, invalidVersion := range invalidVersions {
		if rxPluginVersion.MatchString(invalidVersion) {
			t.Errorf("invalid Version passed test: %s", invalidVersion)
		}
	}
	validVersions := []string{
		"v1.0",
		"v123.123456789",
	}
	for _, validVersion := range validVersions {
		if !rxPluginVersion.MatchString(validVersion) {
			t.Errorf("valid Version failed to test: %s", validVersion)
		}
	}
}

func TestIsValidClusterName(t *testing.T) {
	invalidClusterNames := []string{
		"",
		"✨️",
		"has spaces",
		"random#characters?",
	}
	for _, invalidClusterName := range invalidClusterNames {
		if isValidClusterName(invalidClusterName) {
			t.Errorf("invalid cluster name passed test: %s", invalidClusterName)
		}
	}
	validClusterNames := []string{
		"-",
		"k",
		"k0",
		"ARO3.11",
		"cluster-",
		"_cluster",
		"0cluster",
		"(cluster)",
		"my.cluster",
		"1234567890",
		"osa-testing",
		"ExampleCluster111",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	for _, validClusterName := range validClusterNames {
		if !isValidClusterName(validClusterName) {
			t.Errorf("valid cluster name failed to pass test: %s", validClusterName)
		}
	}
}

func TestIsValidLocation(t *testing.T) {
	invalidLocations := []string{
		"",
		"West US 2",
		"Brazil South",
		"random#characters?",
	}
	for _, invalidLocation := range invalidLocations {
		if isValidLocation(invalidLocation) {
			t.Errorf("invalid location passed test: %s", invalidLocation)
		}
	}
	validLocations := []string{
		"a",
		"eastus",
		"EastUS",
		"westus2",
		"westeurope",
		"canadacentral",
	}
	for _, validLocation := range validLocations {
		if !isValidLocation(validLocation) {
			t.Errorf("valid location failed to pass test: %s", validLocation)
		}
	}
}

func TestIsValidCloudAppHostname(t *testing.T) {
	invalidFqdns := []string{
		"invalid.random.domain",
		"too.long.domain.cloudapp.azure.com",
		"invalid#characters#domain.westus2.cloudapp.azure.com",
		"wronglocation.eastus.cloudapp.azure.com",
		"123.eastus.cloudapp.azure.com",
		"-abc.eastus.cloudapp.azure.com",
		"abcdefghijklmnopqrstuvwxzyabcdefghijklmnopqrstuvwxzyabcdefghijkl.eastus.cloudapp.azure.com",
		"a/b/c.eastus.cloudapp.azure.com",
		".eastus.cloudapp.azure.com",
		"Thisisatest.eastus.cloudapp.azure.com",
	}
	for _, invalidFqdn := range invalidFqdns {
		if isValidCloudAppHostname(invalidFqdn, "westus2") {
			t.Errorf("invalid FQDN passed test: %s", invalidFqdn)
		}
	}
	validFqdns := []string{
		"example.westus2.cloudapp.azure.com",
		"test-dashes.westus2.cloudapp.azure.com",
		"test123.westus2.cloudapp.azure.com",
		"test-123.westus2.cloudapp.azure.com",
	}
	for _, validFqdn := range validFqdns {
		if !isValidCloudAppHostname(validFqdn, "westus2") {
			t.Errorf("Valid FQDN failed to pass test: %s", validFqdn)
		}
	}
}

func TestIsValidIPV4CIDR(t *testing.T) {
	for _, test := range []struct {
		cidr  string
		valid bool
	}{
		{
			cidr: "",
		},
		{
			cidr: "foo",
		},
		{
			cidr: "::/0",
		},
		{
			cidr: "192.168.0.1/24",
		},
		{
			cidr:  "192.168.0.0/24",
			valid: true,
		},
	} {
		valid := isValidIPV4CIDR(test.cidr)
		if valid != test.valid {
			t.Errorf("%s: unexpected result %v", test.cidr, valid)
		}
	}
}

func TestVnetContainsSubnet(t *testing.T) {
	for i, test := range []struct {
		vnetCidr   string
		subnetCidr string
		valid      bool
	}{
		{
			vnetCidr:   "10.0.0.0/16",
			subnetCidr: "192.168.0.0/16",
		},
		{
			vnetCidr:   "10.0.0.0/16",
			subnetCidr: "10.0.0.0/8",
		},
		{
			vnetCidr:   "10.0.0.0/16",
			subnetCidr: "10.0.128.0/15",
		},
		{
			vnetCidr:   "10.0.0.0/8",
			subnetCidr: "10.0.0.0/16",
			valid:      true,
		},
		{
			vnetCidr:   "10.0.0.0/8",
			subnetCidr: "10.0.0.0/8",
			valid:      true,
		},
	} {
		_, vnet, err := net.ParseCIDR(test.vnetCidr)
		if err != nil {
			t.Fatal(err)
		}

		_, subnet, err := net.ParseCIDR(test.subnetCidr)
		if err != nil {
			t.Fatal(err)
		}

		valid := vnetContainsSubnet(vnet, subnet)
		if valid != test.valid {
			t.Errorf("%d: unexpected result %v", i, valid)
		}
	}
}

func TestIsValidAgentPoolHostname(t *testing.T) {
	for _, tt := range []struct {
		hostname string
		valid    bool
	}{
		{
			hostname: "bad",
		},
		{
			hostname: "master-000000",
			valid:    true,
		},
		{
			hostname: "master-00000a",
			valid:    true,
		},
		{
			hostname: "master-00000A",
			valid:    true,
		},
		{
			hostname: "mycompute-000000",
		},
		{
			hostname: "master-bad",
		},
		{
			hostname: "master-inval!",
		},
		{
			hostname: "mycompute-1234567890-000000",
			valid:    true,
		},
		{
			hostname: "mycompute-1234567890-00000z",
			valid:    true,
		},
		{
			hostname: "mycompute-1234567890-00000Z",
			valid:    true,
		},
		{
			hostname: "mycompute-1234-00000Z",
		},
		{
			hostname: "mycompute-1234567890-bad",
		},
		{
			hostname: "mycompute-1234567890-inval!",
		},
		{
			hostname: "master-1234567890-000000",
		},
		{
			hostname: "bad-bad-bad-bad",
		},
	} {
		valid := IsValidAgentPoolHostname(tt.hostname)
		if valid != tt.valid {
			t.Errorf("%s: wanted valid %v, got %v", tt.hostname, tt.valid, valid)
		}
	}
}

func TestIsValidBlobName(t *testing.T) {
	for _, tt := range []struct {
		name  string
		valid bool
	}{
		{
			name:  "123",
			valid: true,
		},
		{
			name:  "abc",
			valid: true,
		},
		{
			name:  "abc-123",
			valid: true,
		},
		{
			name:  "Good",
			valid: true,
		},
		{
			name: "\\bad\\",
		},
		{
			name: "1%",
		},
		{
			name: "\bad",
		},
		{
			name: "foo/c{/}.jpg",
		},
		{
			name: "ba da",
		},
		{
			name:  fmt.Sprintf("backup%s", time.Now().UTC().Format("2006-01-02T15-04-05")),
			valid: true,
		},
	} {
		valid := IsValidBlobName(tt.name)
		if valid != tt.valid {
			t.Errorf("%s: wanted valid %v, got %v", tt.name, tt.valid, valid)
		}
	}
}

func TestIsValidRpmPackageName(t *testing.T) {
	for _, tt := range []struct {
		name  string
		valid bool
	}{
		{
			name:  "nano-2.3.1-10.el7.x86_64",
			valid: true,
		},
		{
			name:  "nano-2.3.1-10",
			valid: true,
		},
		{
			name:  "nano",
			valid: true,
		},
		{
			name:  "n",
			valid: true,
		},
		{
			name:  "Patch_1-+.",
			valid: true,
		},
		{
			name: "package.rpm",
		},
		{
			name: "",
		},
	} {
		valid := isValidRpmPackageName(tt.name)
		if valid != tt.valid {
			t.Errorf("%s: wanted valid %v, got %v", tt.name, tt.valid, valid)
		}
	}
}
