package cluster

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"time"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/roundtrippers"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

func getConsoleClient(cs *api.OpenShiftManagedCluster, testConfig api.TestConfig) wait.SimpleHTTPClient {
	cert := cs.Config.Certificates.OpenShiftConsole.Certs
	pool := x509.NewCertPool()
	pool.AddCert(cert[len(cert)-1])
	tlsConfig := tls.Config{
		RootCAs:    pool,
		ServerName: cs.Properties.PublicHostname,
	}

	return &http.Client{Transport: roundtrippers.HealthCheck(cs.Properties.FQDN, cs.Location, cs.Properties.NetworkProfile.PrivateEndpoint, testConfig, &tlsConfig), Timeout: 10 * time.Second}
}

// HealthCheck function to verify cluster health
func (u *Upgrade) HealthCheck(ctx context.Context) *api.PluginError {
	u.Log.Info("checking developer console health")

	url := "https://" + u.Cs.Properties.PublicHostname + "/console/"
	if u.Cs.Properties.NetworkProfile.PrivateEndpoint != nil {
		url = "https://" + *u.Cs.Properties.NetworkProfile.PrivateEndpoint + "/console/"
	}

	_, err := wait.ForHTTPStatusOk(ctx, u.Log, u.GetConsoleClient(u.Cs, u.TestConfig), url, time.Second)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForConsoleHealth}
	}

	return nil
}

func getAPIServerClient(cs *api.OpenShiftManagedCluster, testConfig api.TestConfig) wait.SimpleHTTPClient {
	pool := x509.NewCertPool()
	pool.AddCert(cs.Config.Certificates.Ca.Cert)
	tlsConfig := tls.Config{
		RootCAs:    pool,
		ServerName: cs.Properties.FQDN,
	}

	return &http.Client{Transport: roundtrippers.HealthCheck(cs.Properties.FQDN, cs.Location, cs.Properties.NetworkProfile.PrivateEndpoint, testConfig, &tlsConfig), Timeout: 10 * time.Second}
}

func (u *Upgrade) WaitForHealthzStatusOk(ctx context.Context) error {
	u.Log.Infof("waiting for API server healthz")
	url := "https://" + u.Cs.Properties.FQDN + "/healthz"
	if u.Cs.Properties.NetworkProfile.PrivateEndpoint != nil {
		url = "https://" + *u.Cs.Properties.NetworkProfile.PrivateEndpoint + "/healthz"
	}
	_, err := wait.ForHTTPStatusOk(ctx, u.Log, u.GetAPIServerClient(u.Cs, u.TestConfig), url, time.Second)
	return err
}
