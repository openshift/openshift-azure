package osa

import (
	"github.com/ghodss/yaml"
	"github.com/jim-minter/azure-helm/pkg/api"
	"github.com/jim-minter/azure-helm/pkg/arm"
	"github.com/jim-minter/azure-helm/pkg/config"
	"github.com/jim-minter/azure-helm/pkg/helm"
	"io/ioutil"
)

type OSA interface {
	Validate() error
	GenerateConfig() (*config.Config, error)
	GenerateHelm() ([]byte, error)
	GenerateARM() ([]byte, error)
	Healthz() error
}

func NewOSA(newManifest, oldManifest []byte) (OSA, error) {
	return &osaManager{
		newManifestBytes: newManifest,
		oldManifestBytes: oldManifest,
	}, nil
}

func NewOSAByPath(newManifest, oldManifest string) (OSA, error) {
	newBytes, err := ioutil.ReadFile(newManifest)
	if err != nil {
		return nil, err
	}

	osa := &osaManager{
		newManifestBytes: newBytes,
	}

	if len(oldManifest) > 0 {
		oldBytes, err := ioutil.ReadFile(newManifest)
		if err != nil {
			return nil, err
		}
		osa.oldManifestBytes = oldBytes
	}
	return osa, nil
}

var _ OSA = &osaManager{}

type osaManager struct {
	newManifestBytes []byte
	oldManifestBytes []byte

	newManifest *api.Manifest
	oldManifest *api.Manifest

	config *config.Config
}

func (m *osaManager) Validate() error {
	// 1.  Unmarshal the manifests
	// 		the manifests will be versioned but by accepting bytes we can hide the versioned
	//		implementation from the callers.
	// 2.  Validate the new manifest with a versioned validate call.
	// 3.  If versioned validate passes convert both manifests to the internal manifest type
	// 4.  Validate the new manifest against the old manifest
	// 5.  Set m.newManifest and m.oldManifest
	// 6.  All further methods can rely on the internal versions

	n, err := unmarshallManifest(m.newManifestBytes)
	if err != nil {
		return err
	}
	o, err := unmarshallManifest(m.oldManifestBytes)
	if err != nil {
		return err
	}

	m.newManifest = n
	m.oldManifest = o
	return nil
}

func (m *osaManager) GenerateConfig() (*config.Config, error) {
	c, err := config.Generate(m.newManifest)
	if err == nil {
		m.config = c
	}
	return c, err
}

func (m *osaManager) GenerateHelm() ([]byte, error) {
	return helm.Generate(m.newManifest, m.config)
}

func (m *osaManager) GenerateARM() ([]byte, error) {
	return arm.Generate(m.newManifest, m.config)
}

func (m *osaManager) Healthz() error {
	return nil
}

func unmarshallManifest(b []byte) (*api.Manifest, error) {
	var m *api.Manifest
	err := yaml.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}

	return m, nil
}
