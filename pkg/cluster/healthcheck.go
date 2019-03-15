package cluster

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

// getHealthCheckHTTPClient returns a specially configured http client.  When
// the client is used to connect to a remote TLS server (e.g.
// openshift.<random>.osadev.cloud), it will in fact dial dialHost (e.g.
// <random>.<location>.cloudapp.azure.com).  It will then negotiate TLS against
// the former address (i.e. openshift.<random>.osadev.cloud), verifying that the
// server certificate presented matches cert.
func getHealthCheckHTTPClient(dialHost string, cert *x509.Certificate) *http.Client {
	pool := x509.NewCertPool()
	pool.AddCert(cert)

	return &http.Client{
		Transport: &http.Transport{
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
		},
		Timeout: 10 * time.Second,
	}
}

func (u *simpleUpgrader) doHealthCheck(ctx context.Context, cli wait.SimpleHTTPClient, uri string, sleepDuration time.Duration) *api.PluginError {
	req, err := http.NewRequest("HEAD", uri, nil)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForConsoleHealth}
	}
	req = req.WithContext(ctx)

	// Wait for the console to be 200 status
	for {
		resp, err := cli.Do(req)
		if err, ok := err.(*url.Error); ok && err.Timeout() {
			time.Sleep(sleepDuration)
			continue
		}
		if err != nil {
			return &api.PluginError{Err: err, Step: api.PluginStepWaitForConsoleHealth}
		}

		switch resp.StatusCode {
		case http.StatusOK:
			return nil
		case http.StatusBadGateway:
			time.Sleep(sleepDuration)
		default:
			err = fmt.Errorf("unexpected error code %d from console", resp.StatusCode)
			return &api.PluginError{Err: err, Step: api.PluginStepWaitForConsoleHealth}
		}
	}
}

// HealthCheck function to verify cluster health
func (u *simpleUpgrader) HealthCheck(ctx context.Context, cs *api.OpenShiftManagedCluster) *api.PluginError {
	u.log.Info("checking developer console health")
	cert := cs.Config.Certificates.OpenShiftConsole.Certs
	err := u.doHealthCheck(ctx, getHealthCheckHTTPClient(cs.Properties.FQDN, cert[len(cert)-1]), "https://"+cs.Properties.PublicHostname+"/console/", time.Second)
	if err != nil {
		return err
	}
	u.log.Info("checking admin console health")
	cert = cs.Config.Certificates.Router.Certs
	return u.doHealthCheck(ctx, getHealthCheckHTTPClient(cs.Properties.RouterProfiles[0].FQDN, cert[len(cert)-1]), "https://console."+cs.Properties.RouterProfiles[0].PublicSubdomain+"/", time.Second)
}

func (u *simpleUpgrader) WaitForHealthzStatusOk(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	_, err := wait.ForHTTPStatusOk(ctx, u.log, u.rt, "https://"+cs.Properties.FQDN+"/healthz")
	return err
}
