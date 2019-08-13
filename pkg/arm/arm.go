package arm

//go:generate go get github.com/golang/mock/mockgen
//go:generate mockgen -destination=../util/mocks/mock_$GOPACKAGE/arm.go -package=mock_$GOPACKAGE -source arm.go
//go:generate gofmt -s -l -w ../util/mocks/mock_$GOPACKAGE/arm.go
//go:generate go get golang.org/x/tools/cmd/goimports
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../util/mocks/mock_$GOPACKAGE/arm.go

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/runtime"
)

func New(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, testConfig api.TestConfig) (api.ARMInterface, error) {
	if runtime.Versions[cs.Config.PluginVersion] != nil {
		fnc := runtime.Versions[cs.Config.PluginVersion]
		return fnc(ctx, log, cs, testConfig), nil
	}
	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}
