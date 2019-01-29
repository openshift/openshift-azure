package cluster

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

func getHealthCheckHTTPClient(cs *api.OpenShiftManagedCluster) *http.Client {
	c := cs.Config
	pool := x509.NewCertPool()
	pool.AddCert(c.Certificates.Ca.Cert)

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
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
	err := u.kubeclient.WaitForInfraServices(ctx)
	if err != nil {
		return err
	}

	u.log.Info("checking developer console health")
	err = u.doHealthCheck(ctx, getHealthCheckHTTPClient(cs), "https://"+cs.Properties.FQDN+"/console/", 10*time.Second)
	return err

	// currently this makes a tcp connection to console.publicsubdomain:443 then
	// issues a GET with Host header console.publicsubdomain

	// in the future when we enable vanity domains, the end user won't have
	// created the publicsubdomain record yet, so this will need to make a tcp
	// connection to console.fqdn:443 with an SNI header set to
	// console.publicsubdomain,then issue a GET with Host header
	// console.publicsubdomain
	// u.log.Info("checking admin console health")
	// return u.doHealthCheck(ctx, getHealthCheckHTTPClient(cs), "https://console."+cs.Properties.RouterProfiles[0].PublicSubdomain, 10*time.Second)
}

func (u *simpleUpgrader) WaitForHealthzStatusOk(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	_, err := wait.ForHTTPStatusOk(ctx, u.log, u.rt, "https://"+cs.Properties.FQDN+"/healthz")
	return err
}
