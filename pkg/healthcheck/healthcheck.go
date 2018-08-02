package healthcheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"time"

	appsclient "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"
	"k8s.io/client-go/tools/clientcmd/api/v1"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/checks"
	"github.com/openshift/openshift-azure/pkg/config"
)

// GetKubeconfigFromV1Config takes a v1 config and returns a kubeconfig
func getKubeconfigFromV1Config(kc *v1.Config) (clientcmd.ClientConfig, error) {
	var c api.Config
	err := latest.Scheme.Convert(kc, &c, nil)
	if err != nil {
		return nil, err
	}

	kubeconfig := clientcmd.NewDefaultClientConfig(c, &clientcmd.ConfigOverrides{})

	return kubeconfig, nil
}

// HealthCheck function to verify cluster health
func HealthCheck(ctx context.Context, cs *acsapi.ContainerService, c *config.Config) error {
	kubeconfig, err := getKubeconfigFromV1Config(c.AdminKubeconfig)
	if err != nil {
		return err
	}

	restconfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return err
	}

	t, err := rest.TransportFor(restconfig)
	if err != nil {
		return err
	}

	// Wait for the healthz to be 200 status
	err = checks.WaitForHTTPStatusOk(ctx, t, restconfig.Host+"/healthz")
	if err != nil {
		return err
	}

	appsclient, err := appsclient.NewForConfig(restconfig)
	if err != nil {
		return err
	}
	// Ensure that the pods in default are healthy
	err = checks.WaitForInfraServices(ctx, appsclient)
	if err != nil {
		return err
	}

	// Wait for the console to be 200 status
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
