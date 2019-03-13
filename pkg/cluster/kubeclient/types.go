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

	"github.com/openshift/openshift-azure/pkg/api"
)

// Kubeclient interface to utility kubenetes functions
type Kubeclient interface {
	BackupCluster(ctx context.Context, backupName string) error
	DrainAndDeleteWorker(ctx context.Context, hostname ComputerName) error
	DeleteMaster(hostname ComputerName) error
	GetControlPlanePods(ctx context.Context) ([]corev1.Pod, error)
	WaitForInfraServices(ctx context.Context) *api.PluginError
	WaitForReadyMaster(ctx context.Context, hostname ComputerName) error
	WaitForReadyWorker(ctx context.Context, hostname ComputerName) error
}

type kubeclient struct {
	client kubernetes.Interface
	log    *logrus.Entry
}

var _ Kubeclient = &kubeclient{}

type ComputerName string

func (hostname ComputerName) toKubernetes() string {
	return strings.ToLower(string(hostname))
}
