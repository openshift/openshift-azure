package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/pkg/transport"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/etcdbackup"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
	"github.com/openshift/openshift-azure/pkg/util/configblob"
	"github.com/openshift/openshift-azure/pkg/util/log"
)

var (
	logLevel    = flag.String("loglevel", "Debug", "Valid values are Debug, Info, Warning, Error")
	blobName    = flag.String("blobname", "", "Name of the blob (without the container)")
	destination = flag.String("destination", "", "Where to place the blob on the filesystem")
	maxBackups  = flag.Int("maxbackups", 6, "Maximum number of backups to keep")
	gitCommit   = "unknown"
)

func myUsage() {
	fmt.Printf("Usage:\n")
	fmt.Printf("\"%s save\" - run the backup with generated blobname \"backup-<timestamp>\"\n", os.Args[0])
	fmt.Printf("\"%s -blobname backup-before-upgrade2 save\" - run the backup with specified blobname \n", os.Args[0])
	fmt.Printf("\"%s -destination /tmp/mybackup.db -blobname backup-before-upgrade2 download\" - download the backup from the named Azure blob\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

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

func main() {
	flag.Usage = myUsage
	flag.Parse()
	logrus.SetLevel(log.SanitizeLogLevel(*logLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	log := logrus.NewEntry(logrus.StandardLogger())
	log.Printf("etcdbackup starting, git commit %s", gitCommit)

	if flag.NArg() != 1 {
		flag.Usage()
	}

	ctx := context.Background()

	cpc, err := cloudprovider.Load("_data/_out/azure.conf")
	if err != nil {
		log.Fatal(errors.Wrap(err, "could not read azure.conf"))
	}

	bsc, err := configblob.GetService(ctx, cpc)
	if err != nil {
		log.Fatal(errors.Wrap(err, "could not find storage account"))
	}
	etcdContainer := bsc.GetContainerReference("etcd")

	etcdcli, err := getEtcdClient()
	if err != nil {
		log.Fatal(errors.Wrap(err, "create etcd client failed"))
	}
	defer etcdcli.Close()
	b := etcdbackup.NewEtcdBackup(log, etcdContainer, etcdcli, *maxBackups)

	switch flag.Arg(0) {
	case "save":
		path := fmt.Sprintf("backup-%s", time.Now().UTC().Format("2006-01-02T15-04-05"))
		if len(*blobName) > 0 {
			path = *blobName
		}
		log.Infof("backing up etcd to %s", path)
		err = b.SaveSnapshot(ctx, path)
		if err != nil {
			// don't override the initial error.
			derr := b.Delete(path)
			if derr != nil {
				log.Errorf("deleting bad backup %s failed with %v", path, derr)
			}
		} else {
			err = b.Prune()
		}
	case "download":
		if len(*destination) == 0 || len(*blobName) == 0 {
			flag.Usage()
		}
		log.Infof("copying backup from %s to %s", *blobName, *destination)
		err = b.Retrieve(*blobName, *destination)
	default:
		flag.Usage()
	}
	if err != nil {
		log.Fatal(err)
	}
}
