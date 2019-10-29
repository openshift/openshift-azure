package startup

import (
	"crypto/x509"
	"reflect"
	"testing"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/test/util/tls"
)

func TestDerivedKubeAndSystemReserved(t *testing.T) {
	tests := []struct {
		cs        api.OpenShiftManagedCluster
		role      api.AgentPoolProfileRole
		wantKR    string
		wantKRErr string
		wantSR    string
		wantSRErr string
	}{
		{
			cs: api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Role:   api.AgentPoolProfileRoleCompute,
							VMSize: api.StandardD4sV3,
						},
					},
				},
			},
			role:   api.AgentPoolProfileRoleCompute,
			wantKR: "cpu=500m,memory=512Mi",
			wantSR: "cpu=500m,memory=512Mi",
		},
		{
			cs: api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Role:   api.AgentPoolProfileRoleInfra,
							VMSize: api.StandardD2sV3,
						},
					},
				}},
			role:   api.AgentPoolProfileRoleInfra,
			wantKR: "cpu=200m,memory=512Mi",
			wantSR: "cpu=200m,memory=512Mi",
		},
		{
			cs: api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Role:   api.AgentPoolProfileRoleMaster,
							VMSize: api.StandardD2sV3,
						},
					},
				},
			},
			role:      api.AgentPoolProfileRoleMaster,
			wantKRErr: "kubereserved not defined for role master",
			wantSR:    "cpu=500m,memory=1Gi",
		},
		{
			cs: api.OpenShiftManagedCluster{
				Properties: api.Properties{
					AgentPoolProfiles: []api.AgentPoolProfile{
						{
							Role:   api.AgentPoolProfileRoleMaster,
							VMSize: api.StandardD4sV3,
						},
					},
				},
			},
			role:      api.AgentPoolProfileRoleMaster,
			wantKRErr: "kubereserved not defined for role master",
			wantSR:    "cpu=1000m,memory=1Gi",
		},
		{
			cs: api.OpenShiftManagedCluster{
				Properties: api.Properties{},
			},
			role:      "anewrole",
			wantKRErr: "role anewrole not found",
			wantSRErr: "role anewrole not found",
		},
	}
	d := &derivedType{}
	for _, tt := range tests {
		got, err := d.KubeReserved(&tt.cs, tt.role)
		if got != tt.wantKR || (err == nil && tt.wantKRErr != "") || (err != nil && err.Error() != tt.wantKRErr) {
			t.Errorf("derived.KubeReserved(%s) = %v, %v: wanted %v, %v", tt.role, got, err, tt.wantKR, tt.wantKRErr)
		}

		got, err = d.SystemReserved(&tt.cs, tt.role)
		if got != tt.wantSR || (err == nil && tt.wantSRErr != "") || (err != nil && err.Error() != tt.wantSRErr) {
			t.Errorf("derived.SystemReserved(%s) = %v, %v: wanted %v, %v", tt.role, got, err, tt.wantSR, tt.wantSRErr)
		}
	}
}

func TestCaBundle(t *testing.T) {
	expected := []*x509.Certificate{
		tls.DummyCertificate,
	}
	cs := api.OpenShiftManagedCluster{
		Config: api.Config{
			Certificates: api.CertificateConfig{
				Ca: api.CertKeyPair{
					Cert: tls.DummyCertificate,
					Key:  tls.DummyPrivateKey,
				},
				OpenShiftConsole: api.CertKeyPairChain{
					Certs: []*x509.Certificate{
						tls.DummyCertificate, tls.DummyCertificate,
					},
				},
				Router: api.CertKeyPairChain{
					Certs: []*x509.Certificate{tls.DummyCertificate},
				},
			},
		},
	}
	d := &derivedType{}
	bundle, err := d.CaBundle(&cs)
	if err != nil {
		t.Errorf("derived.CaBundle() error %v", err)
	}
	if !reflect.DeepEqual(expected, bundle) {
		t.Errorf("derived.CaBundle() = ca-bundle lenght \"%v\", want \"%v\"", len(bundle), len(expected))
	}

}

func TestGetServerFromDNSConf(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
		wantErr bool
	}{
		{
			name:    "expected",
			content: "server=168.63.129.16",
			want:    []string{"168.63.129.16"},
		},
		{
			name:    "bad format",
			content: "168.63.129.16",
			wantErr: true,
		},
		{
			name: "complex",
			want: []string{"168.63.129.16", "1.2.3.4"},
			content: `dns-forward-max=5000
server=168.63.129.16
server=1.2.3.4`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getServerFromDNSConf(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("getServerFromDNSConf() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getServerFromDNSConf() = %v, want %v", got, tt.want)
			}
		})
	}
}
