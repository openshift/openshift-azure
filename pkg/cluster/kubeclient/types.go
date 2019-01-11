package kubeclient

//go:generate go get github.com/golang/mock/gomock
//go:generate go install github.com/golang/mock/mockgen
//go:generate mockgen -destination=../../util/mocks/mock_$GOPACKAGE/types.go  github.com/openshift/openshift-azure/pkg/cluster/$GOPACKAGE Kubeclient
//go:generate gofmt -s -l -w ../../util/mocks/mock_$GOPACKAGE/types.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../util/mocks/mock_$GOPACKAGE/types.go

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

// Kubeclient interface to utility kubenetes functions
type Kubeclient interface {
	DrainAndDeleteWorker(ctx context.Context, computerName ComputerName) error
	DeleteMaster(computerName ComputerName) error
	GetControlPlanePods(ctx context.Context) ([]corev1.Pod, error)
	WaitForInfraServices(ctx context.Context) *api.PluginError
	WaitForReadyMaster(ctx context.Context, computerName ComputerName) error
	WaitForReadyWorker(ctx context.Context, computerName ComputerName) error
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
