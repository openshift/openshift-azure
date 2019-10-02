package privatelink

import "github.com/Azure/go-autorest/version"

// UserAgent returns the UserAgent string to use when sending http.Requests.
func UserAgent() string {
	return "Azure-SDK-For-Go/" + version.Number + " network/2019-06-01"
}

// Version returns the semantic version (see http://semver.org) of the client.
func Version() string {
	return "0.0.0"
}
