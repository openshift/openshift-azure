package config

//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -prefix data data/...
//go:generate gofmt -s -l -w bindata.go

import (
	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util"
)

func generateEtcdConfig(cs *api.OpenShiftManagedCluster) error {

	etcdConfig := api.EtcdConfig{
		AdvertiseUrls:             "https://$(hostname):2379",
		CertFile:                  "/etc/etcd/server.crt",
		ClientCertAuth:            true,
		DataDir:                   "/var/lib/etcd",
		ElectionTimeout:           2500,
		HeartbeatInterval:         500,
		InitialAdvertisePeersUrls: "https://$(hostname):2380",
		InitialCluster:            "master-000000=https://master-000000:2380,master-000001=https://master-000001:2380,master-000002=https://master-000002:2380",
		KeyFile:                   "/etc/etcd/server.key",
		ListenClientsUrls:         "https://0.0.0.0:2379",
		ListenPeerUrls:            "https://0.0.0.0:2380",
		Name:                      "$(hostname)",
		PeerCertFile:              "/etc/etcd/peer.crt",
		PeerClientCertAuth:        true,
		PeerKeyFile:               "/etc/etcd/peer.key",
		PeerTrustedCaFile:         "/etc/etcd/ca.crt",
		QuotaBackendBytes:         4294967296,
		TrustedCaFile:             "/etc/etcd/ca.crt",
	}
	cs.Config.Etcd = etcdConfig
	cfg, err := templateEtcdConfig(cs)
	if err != nil {
		return err
	}
	cs.Config.Etcd.Config = cfg
	return nil

}

func templateEtcdConfig(cs *api.OpenShiftManagedCluster) ([]byte, error) {
	var res []byte
	for _, asset := range AssetNames() {
		b, err := Asset(asset)
		if err != nil {
			return nil, err
		}
		res, err = util.Template(string(b), nil, cs, nil)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}
