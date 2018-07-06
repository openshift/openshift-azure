// Package plugin holds the implementation of a plugin.
package plugin

import (
	"context"
	"io/ioutil"

	"github.com/ghodss/yaml"

	"github.com/jim-minter/azure-helm/pkg/api"
	"github.com/jim-minter/azure-helm/pkg/arm"
	"github.com/jim-minter/azure-helm/pkg/config"
	"github.com/jim-minter/azure-helm/pkg/healthcheck"
	"github.com/jim-minter/azure-helm/pkg/helm"
	"github.com/jim-minter/azure-helm/pkg/tls"
)

type Plugin struct {
	manifestBytes    []byte
	oldManifestBytes []byte

	manifest    *api.Manifest
	oldManifest *api.Manifest

	config *config.Config
}

var _ api.Plugin = &Plugin{}

func NewPlugin(manifestBytes, oldManifestBytes, configBytes []byte) (api.Plugin, error) {
	var config *config.Config
	err := yaml.Unmarshal(configBytes, &config)
	if err != nil {
		return nil, err
	}

	return &Plugin{
		manifestBytes:    manifestBytes,
		oldManifestBytes: oldManifestBytes,
		config:           config,
	}, nil
}

func (p *Plugin) Validate() error {
	// 1.  Unmarshal the manifests
	// 		the manifests will be versioned but by accepting bytes we can hide the versioned
	//		implementation from the callers.
	// 2.  Validate the new manifest with a versioned validate call.
	// 3.  If versioned validate passes convert both manifests to the internal manifest type
	// 4.  Validate the new manifest against the old manifest
	// 5.  Set m.newManifest and m.oldManifest
	// 6.  All further methods can rely on the internal versions

	n, err := unmarshalManifest(p.manifestBytes)
	if err != nil {
		return err
	}
	o, err := unmarshalManifest(p.oldManifestBytes)
	if err != nil {
		return err
	}

	p.manifest = n
	p.oldManifest = o

	return config.Validate(p.manifest, p.oldManifest)
}

func (p *Plugin) GenerateConfig() ([]byte, error) {
	if p.config == nil {
		p.config = &config.Config{}
	}

	err := config.Upgrade(p.manifest, p.config)
	if err != nil {
		return nil, err
	}

	err = config.Generate(p.manifest, p.config)
	if err != nil {
		return nil, err
	}

	b, err := yaml.Marshal(p.config)
	if err != nil {
		return nil, err
	}
	return b, err
}

func (p *Plugin) GenerateHelm() ([]byte, error) {
	return helm.Generate(p.manifest, p.config)
}

func (p *Plugin) GenerateARM() ([]byte, error) {
	return arm.Generate(p.manifest, p.config)
}

func (p *Plugin) HealthCheck(ctx context.Context) error {
	return healthcheck.HealthCheck(ctx, p.manifest, p.config)
}

// WriteHelpers is for development - not part of the external API
func (p *Plugin) WriteHelpers() error {
	b, err := tls.PrivateKeyAsBytes(p.config.SSHKey)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_data/_out/id_rsa", b, 0600)
	if err != nil {
		return err
	}

	b, err = yaml.Marshal(p.config.AdminKubeconfig)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("_data/_out/admin.kubeconfig", b, 0600)
}

func unmarshalManifest(b []byte) (*api.Manifest, error) {
	var m *api.Manifest
	err := yaml.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}

	return m, nil
}
