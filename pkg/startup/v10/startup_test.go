package startup

import (
	"os"
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
)

func TestFilePermissions(t *testing.T) {
	tests := []struct {
		filepath string
		want     os.FileMode
	}{
		{
			filepath: "foo.key",
			want:     os.FileMode(0600),
		},
		{
			filepath: "foo.kubeconfig",
			want:     os.FileMode(0600),
		},
		{
			filepath: "/etc/origin/cloudprovider/azure.conf",
			want:     os.FileMode(0600),
		},
		{
			filepath: "/etc/origin/master/session-secrets.yaml",
			want:     os.FileMode(0600),
		},
		{
			filepath: "foo.config",
			want:     os.FileMode(0644),
		},
		{
			filepath: "/etc/default/mdsd",
			want:     os.FileMode(0644),
		},
		{
			filepath: "/etc/etcd/ca.crt",
			want:     os.FileMode(0644),
		},
		{
			filepath: "/etc/etcd/peer.key",
			want:     os.FileMode(0600),
		},
	}
	for _, tt := range tests {
		t.Run(tt.filepath, func(t *testing.T) {
			s := &startup{}
			if got := s.filePermissions(tt.filepath); got != tt.want {
				t.Errorf("startup.filePermissions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRealFilePathAndContents(t *testing.T) {
	tests := []struct {
		assetPath string
		role      api.AgentPoolProfileRole
		want      string
		wantErr   bool
	}{
		{
			assetPath: "common/etc/default/mdsd",
			role:      api.AgentPoolProfileRoleMaster,
			want:      "/etc/default/mdsd",
		},
		{
			assetPath: "master/etc/etcd/peer.crt",
			role:      api.AgentPoolProfileRoleMaster,
			want:      "/etc/etcd/peer.crt",
		},
		{
			assetPath: "common/etc/default/mdsd",
			role:      api.AgentPoolProfileRoleCompute,
			want:      "/etc/default/mdsd",
		},
		{
			assetPath: "master/etc/etcd/peer.crt",
			role:      api.AgentPoolProfileRoleCompute,
			want:      "",
		},
		{
			assetPath: "worker/etc/origin/node/node-config.yaml",
			role:      api.AgentPoolProfileRoleCompute,
			want:      "/etc/origin/node/node-config.yaml",
		},
		{
			assetPath: "worker/etc/origin/node/node-config.yaml",
			role:      api.AgentPoolProfileRoleMaster,
			want:      "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.assetPath, func(t *testing.T) {
			s := &startup{}
			got, gotContent, err := s.realFilePathAndContents(tt.assetPath, tt.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("startup.realFilePathAndContents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("startup.realFilePathAndContents() got = %v, want %v", got, tt.want)
			}
			if len(gotContent) <= 0 && tt.want != "" {
				t.Errorf("startup.realFilePathAndContents() gotConentLen = %v", len(gotContent))
			}
		})
	}
}
