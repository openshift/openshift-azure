package roundtrippers

import (
	"crypto/tls"
	"net"
	"net/http"

	"github.com/openshift/openshift-azure/pkg/api"
)

// HealthCheck returns a specially configured round tripper.  When the client
// is used to connect to a remote TLS server (e.g.
// openshift.<random>.osadev.cloud), it will in fact dial dialHost (e.g.
// <random>.<location>.cloudapp.azure.com).  It will then negotiate TLS against
// the former address (i.e. openshift.<random>.osadev.cloud), verifying that the
// server certificate presented matches cert.
func HealthCheck(dialHost string, location string, privateEndpoint *string, testConfig api.TestConfig, tlsConfig *tls.Config) http.RoundTripper {
	var c net.Conn
	return &http.Transport{
		DialTLS: func(network, addr string) (net.Conn, error) {
			_, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			// This does the same thing as roundtrippers.NewPrivateEndpoint
			if privateEndpoint != nil {
				c, err = DialHook(network, net.JoinHostPort(*privateEndpoint, port))
				if err != nil {
					return nil, err
				}
			} else {
				c, err = DialHook(network, net.JoinHostPort(dialHost, port))
				if err != nil {
					return nil, err
				}
			}
			return tls.Client(c, tlsConfig), nil
		},
	}
}
