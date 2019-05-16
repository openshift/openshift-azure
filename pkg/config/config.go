package config

//go:generate go get github.com/golang/mock/mockgen
//go:generate mockgen -destination=../util/mocks/mock_$GOPACKAGE/config.go -package=mock_$GOPACKAGE -source config.go
//go:generate gofmt -s -l -w ../util/mocks/mock_$GOPACKAGE/config.go
//go:generate go get golang.org/x/tools/cmd/goimports
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../util/mocks/mock_$GOPACKAGE/config.go

import (
	"fmt"

	"github.com/openshift/openshift-azure/pkg/api"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin"
	v3 "github.com/openshift/openshift-azure/pkg/config/v3"
	v4 "github.com/openshift/openshift-azure/pkg/config/v4"
	v5 "github.com/openshift/openshift-azure/pkg/config/v5"
	v6 "github.com/openshift/openshift-azure/pkg/config/v6"
)

type Interface interface {
	Generate(template *pluginapi.Config, setVersionFields bool) error
	InvalidateSecrets() error
}

// TODO: remove runningUnderTest once v3 is dead
func New(cs *api.OpenShiftManagedCluster, runningUnderTest bool) (Interface, error) {
	switch cs.Config.PluginVersion {
	case "v3.2":
		return v3.New(cs, runningUnderTest), nil
	case "v4.2", "v4.3", "v4.4":
		return v4.New(cs), nil
	case "v5.1":
		return v5.New(cs), nil
	case "v6.0":
		return v6.New(cs), nil
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}
