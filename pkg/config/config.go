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
	v10 "github.com/openshift/openshift-azure/pkg/config/v10"
	v11 "github.com/openshift/openshift-azure/pkg/config/v11"
	v12 "github.com/openshift/openshift-azure/pkg/config/v12"
	v7 "github.com/openshift/openshift-azure/pkg/config/v7"
	v71 "github.com/openshift/openshift-azure/pkg/config/v71"
	v9 "github.com/openshift/openshift-azure/pkg/config/v9"
)

type Interface interface {
	Generate(template *pluginapi.Config, setVersionFields bool) error
	InvalidateSecrets() error
	InvalidateCertificates() error
}

func New(cs *api.OpenShiftManagedCluster) (Interface, error) {
	switch cs.Config.PluginVersion {
	case "v7.0":
		return v7.New(cs), nil
	case "v7.1":
		return v71.New(cs), nil
	case "v9.0":
		return v9.New(cs), nil
	case "v10.0", "v10.1":
		return v10.New(cs), nil
	case "v11.0":
		return v11.New(cs), nil
	case "v12.0":
		return v12.New(cs), nil
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}
