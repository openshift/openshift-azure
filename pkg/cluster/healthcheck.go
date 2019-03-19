package cluster

import (
	"context"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/healthcheck"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

// HealthCheck function to verify cluster health
func (u *simpleUpgrader) HealthCheck(ctx context.Context, cs *api.OpenShiftManagedCluster) *api.PluginError {
	u.log.Info("checking developer console health")
	cert := cs.Config.Certificates.OpenShiftConsole.Certs
	_, err := wait.ForHTTPStatusOk(ctx, u.log, healthcheck.RoundTripper(cs.Properties.FQDN, cert[len(cert)-1]), "https://"+cs.Properties.PublicHostname+"/console/")
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForConsoleHealth}
	}

	return nil
}

func (u *simpleUpgrader) WaitForHealthzStatusOk(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	u.log.Infof("waiting for API server healthz")
	_, err := wait.ForHTTPStatusOk(ctx, u.log, healthcheck.RoundTripper(cs.Properties.FQDN, cs.Config.Certificates.Ca.Cert), "https://"+cs.Properties.FQDN+"/healthz")
	return err
}
