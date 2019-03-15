package kubeclient

//go:generate go get github.com/golang/mock/gomock
//go:generate go install github.com/golang/mock/mockgen
//go:generate mockgen -destination=../../util/mocks/mock_$GOPACKAGE/types.go  github.com/openshift/openshift-azure/pkg/cluster/$GOPACKAGE Kubeclient
//go:generate gofmt -s -l -w ../../util/mocks/mock_$GOPACKAGE/types.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../util/mocks/mock_$GOPACKAGE/types.go

import (
	"context"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// Kubeclient interface to utility kubenetes functions
type Kubeclient interface {
	BackupCluster(ctx context.Context, backupName string) error
	DrainAndDeleteWorker(ctx context.Context, hostname string) error
	DeleteMaster(hostname string) error
	DeletePod(ctx context.Context, namespace, name string) error
	GetControlPlanePods(ctx context.Context) ([]corev1.Pod, error)
	WaitForReadyMaster(ctx context.Context, hostname string) error
	WaitForReadyWorker(ctx context.Context, hostname string) error
	WaitForReadySyncPod(ctx context.Context) error
}

type kubeclient struct {
	client kubernetes.Interface
	log    *logrus.Entry
}

var _ Kubeclient = &kubeclient{}
