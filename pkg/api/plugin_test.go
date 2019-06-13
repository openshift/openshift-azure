// Package api defines the external API for the plugin.
package api

import (
	"context"
	"testing"

	"github.com/openshift/installer/pkg/types"
	azuretypes "github.com/openshift/installer/pkg/types/azure"

	"github.com/openshift/openshift-azure/pkg/util/tls"
	testtls "github.com/openshift/openshift-azure/test/util/tls"
)

func TestPluginValidateConfig(t *testing.T) {
	ctx := context.Background()
	p := &plugin{}
	tests := []struct {
		name    string
		change  func(cfg *types.InstallConfig)
		wantErr bool
	}{
		{
			name: "valid",
			change: func(cfg *types.InstallConfig) {
				cfg.BaseDomain = "basedomain"
				cfg.PullSecret = "{\"auths\":{\"cloud.openshift.com\":{\"auth\":\"foo\",\"email\":\"notme@redhat.com\"}}}"
				cfg.SSHKey, _ = tls.SSHPublicKeyAsString(&testtls.DummyPrivateKey.PublicKey)
				cfg.Platform = types.Platform{
					Azure: &azuretypes.Platform{
						Region:                      "eastus",
						BaseDomainResourceGroupName: "dns",
					},
				}
			},
		},
		{
			name:    "missing azure",
			wantErr: true,
			change: func(cfg *types.InstallConfig) {
				cfg.BaseDomain = "basedomain"
				cfg.PullSecret = "{\"auths\":{\"cloud.openshift.com\":{\"auth\":\"foo\",\"email\":\"notme@redhat.com\"}}}"
				cfg.SSHKey, _ = tls.SSHPublicKeyAsString(&testtls.DummyPrivateKey.PublicKey)
			},
		},
		{
			name:    "missing azure fields",
			wantErr: true,
			change: func(cfg *types.InstallConfig) {
				cfg.BaseDomain = "basedomain"
				cfg.PullSecret = "{\"auths\":{\"cloud.openshift.com\":{\"auth\":\"foo\",\"email\":\"notme@redhat.com\"}}}"
				cfg.SSHKey, _ = tls.SSHPublicKeyAsString(&testtls.DummyPrivateKey.PublicKey)
				cfg.Platform = types.Platform{
					Azure: &azuretypes.Platform{},
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _ := p.GenerateConfig(ctx, "test")
			tt.change(cfg)
			if err := p.ValidateConfig(ctx, cfg); (err != nil) != tt.wantErr {
				t.Errorf("plugin.ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
