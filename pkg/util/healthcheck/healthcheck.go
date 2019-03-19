package healthcheck

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
)

// RoundTripper returns a specially configured round tripper.  When the client
// is used to connect to a remote TLS server (e.g.
// openshift.<random>.osadev.cloud), it will in fact dial dialHost (e.g.
// <random>.<location>.cloudapp.azure.com).  It will then negotiate TLS against
// the former address (i.e. openshift.<random>.osadev.cloud), verifying that the
// server certificate presented matches cert.
func RoundTripper(dialHost string, cert *x509.Certificate) http.RoundTripper {
	pool := x509.NewCertPool()
	pool.AddCert(cert)

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
			return tls.Client(c, &tls.Config{
				RootCAs:    pool,
				ServerName: host,
			}), nil
		},
	}
}
