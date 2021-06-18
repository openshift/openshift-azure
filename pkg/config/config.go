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
	v20 "github.com/openshift/openshift-azure/pkg/config/v20"
	v21 "github.com/openshift/openshift-azure/pkg/config/v21"
	v22 "github.com/openshift/openshift-azure/pkg/config/v22"
	v23 "github.com/openshift/openshift-azure/pkg/config/v23"
)

type Interface interface {
	Generate(template *pluginapi.Config, setVersionFields bool) error
	InvalidateSecrets() error
	InvalidateCertificates() error
}

func New(cs *api.OpenShiftManagedCluster) (Interface, error) {
	switch cs.Config.PluginVersion {
	case "v20.0":
		return v20.New(cs), nil
	case "v21.0":
		return v21.New(cs), nil
	case "v22.0":
		return v22.New(cs), nil
	case "v23.0":
		return v23.New(cs), nil
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}
