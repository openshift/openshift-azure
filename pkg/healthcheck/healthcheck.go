package healthcheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	"github.com/jim-minter/azure-helm/pkg/api"
	"github.com/jim-minter/azure-helm/pkg/config"
)

func HealthCheck(ctx context.Context, m *api.Manifest, c *config.Config) error {
	return waitForConsole(ctx, m, c)
}

func waitForConsole(ctx context.Context, m *api.Manifest, c *config.Config) error {
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

	req, err := http.NewRequest("HEAD", "https://"+m.PublicHostname+"/console/", nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	for {
		resp, err := cli.Do(req)
		if err != nil {
			return err
		}

		switch resp.StatusCode {
		case http.StatusOK:
			return nil
		case http.StatusBadGateway:
			// we expect a 502 when not ready yet -- keep looping
		default:
			return fmt.Errorf("unexpected error code %d from console", resp.StatusCode)
		}

		time.Sleep(time.Second)
	}
}
