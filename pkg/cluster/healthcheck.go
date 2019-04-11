package cluster

import (
	"context"
	"net/http"
	"time"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/healthcheck"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

func getConsoleClient(cs *api.OpenShiftManagedCluster) wait.SimpleHTTPClient {
	cert := cs.Config.Certificates.OpenShiftConsole.Certs
	return &http.Client{Transport: healthcheck.RoundTripper(cs.Properties.FQDN, cert[len(cert)-1]), Timeout: 10 * time.Second}
}

// HealthCheck function to verify cluster health
func (u *SimpleUpgrader) HealthCheck(ctx context.Context) *api.PluginError {
	u.Log.Info("checking developer console health")
	_, err := wait.ForHTTPStatusOk(ctx, u.Log, u.GetConsoleClient(u.Cs), "https://"+u.Cs.Properties.PublicHostname+"/console/", time.Second)
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForConsoleHealth}
	}

	return nil
}

func getAPIServerClient(cs *api.OpenShiftManagedCluster) wait.SimpleHTTPClient {
	return &http.Client{Transport: healthcheck.RoundTripper(cs.Properties.FQDN, cs.Config.Certificates.Ca.Cert), Timeout: 10 * time.Second}
}

func (u *SimpleUpgrader) WaitForHealthzStatusOk(ctx context.Context) error {
	u.Log.Infof("waiting for API server healthz")
	_, err := wait.ForHTTPStatusOk(ctx, u.Log, u.GetAPIServerClient(u.Cs), "https://"+u.Cs.Properties.FQDN+"/healthz", time.Second)
	return err
}
