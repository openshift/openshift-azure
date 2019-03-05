package validate

import "testing"

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
		if imageVersion.MatchString(invalidVersion) {
			t.Errorf("invalid Version passed test: %s", invalidVersion)
		}
	}
	validVersions := []string{
		"123.1.12345678",
		"123.123.12345678",
		"123.1234.12345678",
	}
	for _, validVersion := range validVersions {
		if !imageVersion.MatchString(validVersion) {
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
		if pluginVersion.MatchString(invalidVersion) {
			t.Errorf("invalid Version passed test: %s", invalidVersion)
		}
	}
	validVersions := []string{
		"v1.0",
		"v123.123456789",
	}
	for _, validVersion := range validVersions {
		if !pluginVersion.MatchString(validVersion) {
			t.Errorf("valid Version failed to test: %s", validVersion)
		}
	}
}
