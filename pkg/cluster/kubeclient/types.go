package kubeclient

//go:generate go get github.com/golang/mock/mockgen
//go:generate mockgen -destination=../../util/mocks/mock_$GOPACKAGE/types.go  github.com/openshift/openshift-azure/pkg/cluster/$GOPACKAGE Interface
//go:generate gofmt -s -l -w ../../util/mocks/mock_$GOPACKAGE/types.go
//go:generate go get golang.org/x/tools/cmd/goimports
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../util/mocks/mock_$GOPACKAGE/types.go

import (
	"context"

	security "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/openshift/openshift-azure/pkg/api"
)

// Interface interface to utility kubenetes functions
type Interface interface {
	BackupCluster(ctx context.Context, backupName string) error
	DrainAndDeleteWorker(ctx context.Context, hostname string) error
	DeleteMaster(hostname string) error
	EnsureSyncPod(ctx context.Context, syncImage string, hash []byte) error
	GetControlPlanePods(ctx context.Context) ([]corev1.Pod, error)
	WaitForReadyMaster(ctx context.Context, hostname string) error
	WaitForReadyWorker(ctx context.Context, hostname string) error
	WaitForReadySyncPod(ctx context.Context) error
}

type Kubeclientset struct {
	Client kubernetes.Interface
	Seccli security.Interface
	Log    *logrus.Entry

	// for internal reuse
	restconfig        *rest.Config
	disableKeepAlives bool

	// for test only
	testConfig api.TestConfig
}

var _ Interface = &Kubeclientset{}
