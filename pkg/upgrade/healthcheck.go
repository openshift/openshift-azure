package upgrade

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

// HealthCheck function to verify cluster health
func (hc *simpleUpgrader) HealthCheck(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	// Wait for the console to be 200 status
	return hc.waitForConsole(ctx, cs)
}

// WaitForInfraServices verifies daemonsets, statefulsets
func (hc *simpleUpgrader) WaitForInfraServices(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	kc, err := managedcluster.ClientsetFromV1ConfigAndWait(ctx, cs.Config.AdminKubeconfig)
	if err != nil {
		return err
	}
	return WaitForInfraServices(ctx, kc)
}

func (hc *simpleUpgrader) waitForConsole(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	log.Info("checking console health")
	c := cs.Config
	pool := x509.NewCertPool()
	pool.AddCert(c.Certificates.Ca.Cert)

	cli := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
			},
		},
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("HEAD", "https://"+cs.Properties.FQDN+"/console/", nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	for {
		resp, err := cli.Do(req)
		if err, ok := err.(*url.Error); ok && err.Timeout() {
			time.Sleep(10 * time.Second)
			continue
		}
		if err != nil {
			return err
		}

		switch resp.StatusCode {
		case http.StatusOK:
			log.Info("OK")
			return nil
		case http.StatusBadGateway:
			time.Sleep(10 * time.Second)
		default:
			return fmt.Errorf("unexpected error code %d from console", resp.StatusCode)
		}
	}
}
