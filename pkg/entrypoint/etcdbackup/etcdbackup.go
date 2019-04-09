package etcdbackup

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/pkg/transport"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/cluster"
	"github.com/openshift/openshift-azure/pkg/etcdbackup"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
	"github.com/openshift/openshift-azure/pkg/util/configblob"
	"github.com/openshift/openshift-azure/pkg/util/log"
)

func getEtcdClient() (*clientv3.Client, error) {
	tlsInfo := transport.TLSInfo{
		CertFile:      "/etc/origin/master/master.etcd-client.crt",
		KeyFile:       "/etc/origin/master/master.etcd-client.key",
		TrustedCAFile: "/etc/origin/master/master.etcd-ca.crt",
	}
	etcdTLSConfig, err := tlsInfo.ClientConfig()
	if err != nil {
		return nil, err
	}
	cfg := clientv3.Config{
		TLS: etcdTLSConfig,
		Endpoints: []string{
			"https://master-000000:2379",
			"https://master-000001:2379",
			"https://master-000002:2379"},
	}
	return clientv3.New(cfg)
}

func start(cfg *Config) error {
	ctx := context.Background()
	logrus.SetLevel(log.SanitizeLogLevel(cfg.LogLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log := logrus.NewEntry(logrus.StandardLogger())

	log.Print("etcdbackup starting")

	cpc, err := cloudprovider.Load("_data/_out/azure.conf")
	if err != nil {
		return fmt.Errorf("could not read azure.conf", err)
	}

	bsc, err := configblob.GetService(ctx, cpc)
	if err != nil {
		return fmt.Errorf("could not find storage account", err)
	}
	etcdContainer := bsc.GetContainerReference(cluster.EtcdBackupContainerName)

	etcdcli, err := getEtcdClient()
	if err != nil {
		return fmt.Errorf("create etcd client failed", err)
	}
	defer etcdcli.Close()
	b := etcdbackup.NewEtcdBackup(log, etcdContainer, etcdcli, cfg.MaxBackups)

	switch cfg.Action {
	case "save":
		path := fmt.Sprintf("backup-%s", time.Now().UTC().Format("2006-01-02T15-04-05"))
		if len(cfg.BlobName) > 0 {
			path = cfg.BlobName
		}
		log.Infof("backing up etcd to %s", path)
		err = b.SaveSnapshot(ctx, path)
		if err != nil {
			// don't override the initial error.
			derr := b.Delete(path)
			if derr != nil {
				return fmt.Errorf("deleting bad backup %s failed with %v", path, derr)
			}
		} else {
			err = b.Prune()
		}
	case "download":
		if len(cfg.Destination) == 0 || len(cfg.BlobName) == 0 {
			return fmt.Errorf("destination and blobName can't be empty")
		}
		log.Infof("copying backup from %s to %s", cfg.BlobName, cfg.Destination)
		err = b.Retrieve(cfg.BlobName, cfg.Destination)
	default:
		flag.Usage()
	}
	return err
}
