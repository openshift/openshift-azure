package arm

//go:generate go get github.com/golang/mock/mockgen
//go:generate mockgen -destination=../util/mocks/mock_$GOPACKAGE/arm.go -package=mock_$GOPACKAGE -source arm.go
//go:generate gofmt -s -l -w ../util/mocks/mock_$GOPACKAGE/arm.go
//go:generate go get golang.org/x/tools/cmd/goimports
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../util/mocks/mock_$GOPACKAGE/arm.go

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	v6 "github.com/openshift/openshift-azure/pkg/arm/v6"
	v7 "github.com/openshift/openshift-azure/pkg/arm/v7"
	v8 "github.com/openshift/openshift-azure/pkg/arm/v8"
)

type Interface interface {
	Generate(ctx context.Context, backupBlob string, isUpdate bool, suffix string) (map[string]interface{}, error)
	Vmss(app *api.AgentPoolProfile, backupBlob, suffix string) (*compute.VirtualMachineScaleSet, error)
	Hash(app *api.AgentPoolProfile) ([]byte, error)
}

func New(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, testConfig api.TestConfig) (Interface, error) {
	switch cs.Config.PluginVersion {
	case "v6.0":
		return v6.New(ctx, log, cs, testConfig), nil
	case "v7.0":
		return v7.New(ctx, log, cs, testConfig), nil
	case "v8.0":
		return v8.New(ctx, log, cs, testConfig), nil
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}
