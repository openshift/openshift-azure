package client

import (
	"context"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"
	"k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/openshift/openshift-azure/pkg/checks"
)

// NewKubernetesClientset returns a kubernetes ClientSet for the given
// kubeconfig
func NewKubernetesClientset(ctx context.Context, config *v1.Config) (*kubernetes.Clientset, error) {
	kubeconfig, err := getKubeconfigFromV1Config(config)
	if err != nil {
		return nil, err
	}

	restconfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	t, err := rest.TransportFor(restconfig)
	if err != nil {
		return nil, err
	}

	// Wait for the healthz to be 200 status
	err = checks.WaitForHTTPStatusOk(ctx, t, restconfig.Host+"/healthz")
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(restconfig)
}

// getKubeconfigFromV1Config takes a v1 config and returns a kubeconfig
func getKubeconfigFromV1Config(kc *v1.Config) (clientcmd.ClientConfig, error) {
	var c api.Config

	if err := latest.Scheme.Convert(kc, &c, nil); err != nil {
		return nil, err
	}

	return clientcmd.NewDefaultClientConfig(c, &clientcmd.ConfigOverrides{}), nil
}
