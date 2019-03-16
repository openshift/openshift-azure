package cluster

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

// getHealthCheckRoundTripper returns a specially configured round tripper.
// When the client is used to connect to a remote TLS server (e.g.
// openshift.<random>.osadev.cloud), it will in fact dial dialHost (e.g.
// <random>.<location>.cloudapp.azure.com).  It will then negotiate TLS against
// the former address (i.e. openshift.<random>.osadev.cloud), verifying that the
// server certificate presented matches cert.
func getHealthCheckRoundTripper(dialHost string, cert *x509.Certificate) http.RoundTripper {
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

// HealthCheck function to verify cluster health
func (u *simpleUpgrader) HealthCheck(ctx context.Context, cs *api.OpenShiftManagedCluster) *api.PluginError {
	u.log.Info("checking developer console health")
	cert := cs.Config.Certificates.OpenShiftConsole.Certs
	_, err := wait.ForHTTPStatusOk(ctx, u.log, getHealthCheckRoundTripper(cs.Properties.FQDN, cert[len(cert)-1]), "https://"+cs.Properties.PublicHostname+"/console/")
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForConsoleHealth}
	}

	u.log.Info("checking admin console health")
	cert = cs.Config.Certificates.Router.Certs
	_, err = wait.ForHTTPStatusOk(ctx, u.log, getHealthCheckRoundTripper(cs.Properties.RouterProfiles[0].FQDN, cert[len(cert)-1]), "https://console."+cs.Properties.RouterProfiles[0].PublicSubdomain+"/")
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForAdminConsoleHealth}
	}
	return nil
}

func (u *simpleUpgrader) WaitForHealthzStatusOk(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	_, err := wait.ForHTTPStatusOk(ctx, u.log, getHealthCheckRoundTripper(cs.Properties.FQDN, cs.Config.Certificates.Ca.Cert), "https://"+cs.Properties.FQDN+"/healthz")
	return err
}
