package roundtrippers

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/openshift/openshift-azure/pkg/api"
)

// HealthCheck returns a specially configured round tripper.  When the client
// is used to connect to a remote TLS server (e.g.
// openshift.<random>.osadev.cloud), it will in fact dial dialHost (e.g.
// <random>.<location>.cloudapp.azure.com).  It will then negotiate TLS against
// the former address (i.e. openshift.<random>.osadev.cloud), verifying that the
// server certificate presented matches cert.
func HealthCheck(dialHost string, location string, privateEndpoint *string, testConfig api.TestConfig, tlsConfig *tls.Config) http.RoundTripper {
	if testConfig.RunningUnderTest && privateEndpoint != nil {
		tlsConfig.Certificates = append(tlsConfig.Certificates, testConfig.ProxyCertificate)
		tlsConfig.InsecureSkipVerify = true
		proxyURL := os.Getenv(fmt.Sprintf("PROXYURL_%s", strings.ToUpper(location)))

		return &http.Transport{
			Proxy: func(*http.Request) (*url.URL, error) {
				return url.Parse(fmt.Sprintf("https://%s:8443/", proxyURL))
			},
			TLSClientConfig:     tlsConfig,
			TLSHandshakeTimeout: 10 * time.Second,
		}
	}

	return &http.Transport{
		DialTLS: func(network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			c, err := net.Dial(network, net.JoinHostPort(dialHost, port))
			if err != nil {
				return nil, err
			}
			tlsConfig.ServerName = host
			return tls.Client(c, tlsConfig), nil
		},
	}
}
