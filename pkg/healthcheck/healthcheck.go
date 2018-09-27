package healthcheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/openshift/openshift-azure/pkg/upgrade"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"

	"github.com/sirupsen/logrus"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
)

type HealthChecker interface {
	HealthCheck(ctx context.Context, cs *acsapi.OpenShiftManagedCluster) error
}

type simpleHealthChecker struct {
	pluginConfig acsapi.PluginConfig
}

var _ HealthChecker = &simpleHealthChecker{}

// NewSimpleHealthChecker create a new HealthChecker
func NewSimpleHealthChecker(entry *logrus.Entry, pluginConfig acsapi.PluginConfig) HealthChecker {
	log.New(entry)
	return &simpleHealthChecker{pluginConfig: pluginConfig}
}

// HealthCheck function to verify cluster health
func (hc *simpleHealthChecker) HealthCheck(ctx context.Context, cs *acsapi.OpenShiftManagedCluster) error {
	kc, err := managedcluster.ClientSetFromV1Config(ctx, cs.Config.AdminKubeconfig)
	if err != nil {
		return err
	}

	// ensure that all nodes are ready
	err = upgrade.WaitForNodes(ctx, cs, kc)
	if err != nil {
		return err
	}

	// Wait for infrastructure services to be healthy
	err = upgrade.WaitForInfraServices(ctx, kc)
	if err != nil {
		return err
	}

	// Wait for the console to be 200 status
	return hc.waitForConsole(ctx, cs)
}

func (hc *simpleHealthChecker) waitForConsole(ctx context.Context, cs *acsapi.OpenShiftManagedCluster) error {
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
