package config

//go:generate go get github.com/golang/mock/mockgen
//go:generate mockgen -destination=../util/mocks/mock_$GOPACKAGE/types.go -package=mock_$GOPACKAGE -source types.go
//go:generate gofmt -s -l -w ../util/mocks/mock_$GOPACKAGE/types.go
//go:generate go get golang.org/x/tools/cmd/goimports
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../util/mocks/mock_$GOPACKAGE/types.go

import (
	"github.com/openshift/openshift-azure/pkg/api"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin/api"
)

// Generator is an interface for sharing the cluster and plugin configs
type Generator interface {
	Generate(template *pluginapi.Config) error
	InvalidateSecrets() error
}

type simpleGenerator struct {
	cs *api.OpenShiftManagedCluster
}

var _ Generator = &simpleGenerator{}

// NewSimpleGenerator creates a struct to hold both the cluster and plugin configs
func NewSimpleGenerator(cs *api.OpenShiftManagedCluster) Generator {
	return &simpleGenerator{cs: cs}
}
