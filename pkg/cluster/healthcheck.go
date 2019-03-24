package cluster

import (
	"context"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/healthcheck"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

// HealthCheck function to verify cluster health
func (u *simpleUpgrader) HealthCheck(ctx context.Context) *api.PluginError {
	u.log.Info("checking developer console health")
	cert := u.cs.Config.Certificates.OpenShiftConsole.Certs
	_, err := wait.ForHTTPStatusOk(ctx, u.log, healthcheck.RoundTripper(u.cs.Properties.FQDN, cert[len(cert)-1]), "https://"+u.cs.Properties.PublicHostname+"/console/")
	if err != nil {
		return &api.PluginError{Err: err, Step: api.PluginStepWaitForConsoleHealth}
	}

	return nil
}

func (u *simpleUpgrader) WaitForHealthzStatusOk(ctx context.Context) error {
	u.log.Infof("waiting for API server healthz")
	_, err := wait.ForHTTPStatusOk(ctx, u.log, healthcheck.RoundTripper(u.cs.Properties.FQDN, u.cs.Config.Certificates.Ca.Cert), "https://"+u.cs.Properties.FQDN+"/healthz")
	return err
}
