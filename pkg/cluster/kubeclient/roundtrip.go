package kubeclient

import (
	security "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	v1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
	"github.com/openshift/openshift-azure/pkg/util/roundtrippers"
)

// NewKubeclient creates a new kubeclient.  Only called in RP/fakerp context
func NewKubeclient(log *logrus.Entry, config *v1.Config, cs *api.OpenShiftManagedCluster, disableKeepAlives bool) (Interface, error) {
	restconfig, err := NewRestConfig(log, config, cs, disableKeepAlives)
	if err != nil {
		return nil, err
	}

	cli, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	seccli, err := security.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	return &Kubeclientset{
		Log:    log,
		Client: cli,
		Seccli: seccli,
	}, nil
}

// NewRestConfig returns restconfig, based on configuration.  Called in
// RP/fakerp and e2e test context
func NewRestConfig(log *logrus.Entry, config *v1.Config, cs *api.OpenShiftManagedCluster, disableKeepAlives bool) (*rest.Config, error) {
	restconfig, err := managedcluster.RestConfigFromV1Config(config)
	if err != nil {
		return nil, err
	}

	restconfig.WrapTransport = roundtrippers.NewRetryingRoundTripper(log, cs.Location, cs.Properties.NetworkProfile.PrivateEndpoint, disableKeepAlives)

	return restconfig, nil
}
