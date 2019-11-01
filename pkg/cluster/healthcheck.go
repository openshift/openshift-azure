package cluster

import (
	"context"
	"crypto/x509"
	"net/http"
	"time"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/roundtrippers"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

func getConsoleClient(cs *api.OpenShiftManagedCluster) wait.SimpleHTTPClient {
	var cert *x509.Certificate
	if cs.Properties.PrivateAPIServer {
		// private cluster does not have the named serving cert
		cert = cs.Config.Certificates.Ca.Cert
	} else {
		cert = cs.Config.Certificates.OpenShiftConsole.Certs[len(cs.Config.Certificates.OpenShiftConsole.Certs)-1]
	}
	return &http.Client{Transport: roundtrippers.HealthCheck(cs.Properties.FQDN, cert, cs.Location, cs.Properties.NetworkProfile.PrivateEndpoint), Timeout: 10 * time.Second}
}

// HealthCheck function to verify cluster health
func (u *Upgrade) HealthCheck(ctx context.Context) *api.PluginError {
	u.Log.Info("checking developer console health")
	_, err := wait.ForHTTPStatusOk(ctx, u.Log, u.GetConsoleClient(u.Cs), "https://"+u.Cs.Properties.PublicHostname+"/console/", time.Second)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForConsoleHealth}
	}

	return nil
}

func getAPIServerClient(cs *api.OpenShiftManagedCluster) wait.SimpleHTTPClient {
	return &http.Client{Transport: roundtrippers.HealthCheck(cs.Properties.FQDN, cs.Config.Certificates.Ca.Cert, cs.Location, cs.Properties.NetworkProfile.PrivateEndpoint), Timeout: 10 * time.Second}
}

func (u *Upgrade) WaitForHealthzStatusOk(ctx context.Context) error {
	u.Log.Infof("waiting for API server healthz")
	_, err := wait.ForHTTPStatusOk(ctx, u.Log, u.GetAPIServerClient(u.Cs), "https://"+u.Cs.Properties.FQDN+"/healthz", time.Second)
	return err
}
