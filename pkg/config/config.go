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
	v5 "github.com/openshift/openshift-azure/pkg/config/v5"
	v6 "github.com/openshift/openshift-azure/pkg/config/v6"
	v7 "github.com/openshift/openshift-azure/pkg/config/v7"
)

type Interface interface {
	Generate(template *pluginapi.Config, setVersionFields bool) error
	InvalidateSecrets() error
	InvalidateCertificates() error
}

func New(cs *api.OpenShiftManagedCluster) (Interface, error) {
	switch cs.Config.PluginVersion {
	case "v5.1", "v5.2":
		return v5.New(cs), nil
	case "v6.0":
		return v6.New(cs), nil
	case "v7.0":
		return v7.New(cs), nil
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}
