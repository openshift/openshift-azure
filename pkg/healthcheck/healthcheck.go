package healthcheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/checks"
	"github.com/openshift/openshift-azure/pkg/log"
)

type HealthChecker interface {
	HealthCheck(ctx context.Context, cs *acsapi.OpenShiftManagedCluster) error
}

type simpleHealthChecker struct{}

var _ HealthChecker = &simpleHealthChecker{}

func NewSimpleHealthChecker(entry *logrus.Entry) HealthChecker {
	log.New(entry)
	return &simpleHealthChecker{}
}

// HealthCheck function to verify cluster health
func (hc *simpleHealthChecker) HealthCheck(ctx context.Context, cs *acsapi.OpenShiftManagedCluster) error {
	kc, err := newKubernetesClientset(ctx, cs.Config.AdminKubeconfig)
	if err != nil {
		return err
	}

	err = ensureSyncPod(kc, cs)
	if err != nil {
		return err
	}

	anc, err := newAzureClients(ctx, cs)
	if err != nil {
		return err
	}

	// Ensure that the pods in default are healthy
	err = checks.WaitForInfraServices(ctx, kc.AppsV1())
	if err != nil {
		return err
	}

	// Check if FQDN's in the config matches what we got allocated in the cloud
	err = checks.CheckDNS(ctx, anc.eip, anc.lb, cs)
	if err != nil {
		return err
	}

	// Wait for the console to be 200 status
	return hc.waitForConsole(ctx, cs)
}

func (hc *simpleHealthChecker) waitForConsole(ctx context.Context, cs *acsapi.OpenShiftManagedCluster) error {
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
