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
	v20 "github.com/openshift/openshift-azure/pkg/arm/v20"
	v21 "github.com/openshift/openshift-azure/pkg/arm/v21"
	v22 "github.com/openshift/openshift-azure/pkg/arm/v22"
	v23 "github.com/openshift/openshift-azure/pkg/arm/v23"
)

type Interface interface {
	Generate(ctx context.Context, backupBlob string, isUpdate bool, suffix string) (map[string]interface{}, error)
	Vmss(app *api.AgentPoolProfile, backupBlob, suffix string) (*compute.VirtualMachineScaleSet, error)
	Hash(app *api.AgentPoolProfile) ([]byte, error)
}

func New(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, testConfig api.TestConfig) (Interface, error) {
	switch cs.Config.PluginVersion {
	case "v20.0":
		return v20.New(ctx, log, cs, testConfig), nil
	case "v21.0":
		return v21.New(ctx, log, cs, testConfig), nil
	case "v22.0":
		return v22.New(ctx, log, cs, testConfig), nil
	case "v23.0":
		return v23.New(ctx, log, cs, testConfig), nil
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}
