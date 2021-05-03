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
	v15 "github.com/openshift/openshift-azure/pkg/config/v15"
	v16 "github.com/openshift/openshift-azure/pkg/config/v16"
	v17 "github.com/openshift/openshift-azure/pkg/config/v17"
	v19 "github.com/openshift/openshift-azure/pkg/config/v19"
	v20 "github.com/openshift/openshift-azure/pkg/config/v20"
	v21 "github.com/openshift/openshift-azure/pkg/config/v21"
	v22 "github.com/openshift/openshift-azure/pkg/config/v22"
)

type Interface interface {
	Generate(template *pluginapi.Config, setVersionFields bool) error
	InvalidateSecrets() error
	InvalidateCertificates() error
}

func New(cs *api.OpenShiftManagedCluster) (Interface, error) {
	switch cs.Config.PluginVersion {
	case "v15.0":
		return v15.New(cs), nil
	case "v16.0", "v16.1":
		return v16.New(cs), nil
	case "v17.0":
		return v17.New(cs), nil
	case "v19.0":
		return v19.New(cs), nil
	case "v20.0":
		return v20.New(cs), nil
	case "v21.0":
		return v21.New(cs), nil
	case "v22.0":
		return v22.New(cs), nil
	}

	return nil, fmt.Errorf("version %q not found", cs.Config.PluginVersion)
}
