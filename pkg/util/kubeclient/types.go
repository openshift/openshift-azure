package kubeclient

//go:generate go get github.com/golang/mock/gomock
//go:generate go install github.com/golang/mock/mockgen
//go:generate mockgen -destination=../mocks/mock_$GOPACKAGE/types.go  github.com/openshift/openshift-azure/pkg/util/$GOPACKAGE Kubeclient
//go:generate gofmt -s -l -w ../mocks/mock_$GOPACKAGE/types.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../mocks/mock_$GOPACKAGE/types.go

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

// Kubeclient interface to utility kubenetes functions
type Kubeclient interface {
	Drain(ctx context.Context, role api.AgentPoolProfileRole, computerName ComputerName) error
	WaitForInfraServices(ctx context.Context) *api.PluginError
	WaitForReady(ctx context.Context, role api.AgentPoolProfileRole, computerName ComputerName) error
	MasterIsReady(computerName ComputerName) (bool, error)
}

type kubeclient struct {
	pluginConfig api.PluginConfig
	client       kubernetes.Interface
	log          *logrus.Entry
}

var _ Kubeclient = &kubeclient{}

// NewKubeclient creates a new kubelient instance
func NewKubeclient(log *logrus.Entry, config *v1.Config, pluginConfig *api.PluginConfig) (Kubeclient, error) {
	restconfig, err := managedcluster.RestConfigFromV1Config(config)
	if err != nil {
		return nil, err
	}
	cli, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	return &kubeclient{
		pluginConfig: *pluginConfig,
		log:          log,
		client:       cli,
	}, nil

}

type ComputerName string

func (computerName ComputerName) toKubernetes() string {
	return strings.ToLower(string(computerName))
}
