package healthcheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"time"

	acsapi "github.com/Azure/acs-engine/pkg/api"

	"github.com/jim-minter/azure-helm/pkg/config"
)

func HealthCheck(ctx context.Context, cs *acsapi.ContainerService, c *config.Config) error {
	return waitForConsole(ctx, cs, c)
}

func waitForConsole(ctx context.Context, cs *acsapi.ContainerService, c *config.Config) error {
	pool := x509.NewCertPool()
	pool.AddCert(c.CaCert)

	cli := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
			},
		},
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("HEAD", "https://"+cs.Properties.OrchestratorProfile.OpenShiftConfig.PublicHostname+"/console/", nil)
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
			return nil
		case http.StatusBadGateway:
			time.Sleep(10 * time.Second)
		default:
			return fmt.Errorf("unexpected error code %d from console", resp.StatusCode)
		}
	}
}
