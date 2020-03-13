package startup

//go:generate go get github.com/golang/mock/mockgen
//go:generate mockgen -destination=../util/mocks/mock_$GOPACKAGE/startup.go -package=mock_$GOPACKAGE -source startup.go
//go:generate gofmt -s -l -w ../util/mocks/mock_$GOPACKAGE/startup.go
//go:generate go get golang.org/x/tools/cmd/goimports
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../util/mocks/mock_$GOPACKAGE/startup.go

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	v14 "github.com/openshift/openshift-azure/pkg/startup/v14"
	v15 "github.com/openshift/openshift-azure/pkg/startup/v15"
	v16 "github.com/openshift/openshift-azure/pkg/startup/v16"
	v17 "github.com/openshift/openshift-azure/pkg/startup/v17"
)

// Interface is a singleton interface to interact with startup
type Interface interface {
	WriteFiles(ctx context.Context) error
	Hash(role api.AgentPoolProfileRole) ([]byte, error)
	GetWorkerCs() *api.OpenShiftManagedCluster
	WriteSearchDomain(ctx context.Context, log *logrus.Entry) error
}

// New returns a new startup Interface according to the cluster version running
func New(log *logrus.Entry, cs *api.OpenShiftManagedCluster, testConfig api.TestConfig) (Interface, error) {
	switch cs.Config.PluginVersion {
	case "v14.1":
		return v14.New(log, cs, testConfig), nil
	case "v15.0":
		return v15.New(log, cs, testConfig), nil
	case "v16.0":
		return v16.New(log, cs, testConfig), nil
	case "v17.0":
		return v17.New(log, cs, testConfig), nil
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}
