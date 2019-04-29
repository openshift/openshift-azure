package etcdbackup

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"time"

	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/coreos/etcd/clientv3"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

// EtcdBackup used to perform backup maintenance
type EtcdBackup interface {
	SaveSnapshot(ctx context.Context, bname string) error
	Retrieve(srcBlob, destPath string) error
	Delete(name string) error
	Prune() error
}

type etcdBackup struct {
	etcdContainer storage.Container
	etcdClient    *clientv3.Client
	maxBackups    int
	log           *logrus.Entry
}

var _ EtcdBackup = &etcdBackup{}

// NewEtcdBackup create a new instance
func NewEtcdBackup(log *logrus.Entry, etcdContainer storage.Container, etcdClient *clientv3.Client, maxBackups int) EtcdBackup {
	eb := etcdBackup{
		etcdContainer: etcdContainer,
		etcdClient:    etcdClient,
		maxBackups:    maxBackups,
		log:           log,
	}
	return &eb
}

func (b *etcdBackup) SaveSnapshot(ctx context.Context, bname string) error {
	b.log.Infof("Creating Snapshot")
	rc, err := b.etcdClient.Snapshot(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to receive snapshot")
	}
	defer rc.Close()

	b.log.Infof("Creating New Blob")

	blob := b.etcdContainer.GetBlobReference(bname)
	err = blob.CreateBlockBlob(&azstorage.PutBlobOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to create blob %v", bname)
	}

	bw, err := newBlobWriter(blob)
	if err != nil {
		return errors.Wrap(err, "failed to create blob")
	}

	b.log.Infof("Copying blocks to blob")
	bufferedBw := bufio.NewWriterSize(bw, 1024*1024)
	_, err = io.Copy(bufferedBw, rc)
	if err != nil {
		return errors.Wrap(err, "failed to copy blob blocks")
	}
	err = bufferedBw.Flush()
	if err != nil {
		return errors.Wrap(err, "failed to flush to blob")
	}
	// Note: do not use defer, as we want to keep the err
	err = bw.Close()
	if err != nil {
		return errors.Wrap(err, "BlobWriter.Close")
	}
	b.log.Infof("Snapshot saved to blob storage")
	return nil
}

func (b *etcdBackup) Prune() error {
	params := azstorage.ListBlobsParameters{
		Prefix: "backup-",
	}
	blobs, err := b.etcdContainer.ListBlobs(params)
	if err != nil {
		return err
	}
	toDelete := 0
	if len(blobs.Blobs) > b.maxBackups {
		toDelete = len(blobs.Blobs) - b.maxBackups
	} else {
		return nil
	}
	// this should sort oldest first
	sort.Slice(blobs.Blobs, func(i, j int) bool {
		return time.Time(blobs.Blobs[i].Properties.LastModified).Before(time.Time(blobs.Blobs[j].Properties.LastModified))
	})
	for _, blob := range blobs.Blobs[:toDelete] {
		b.log.Infof("pruning blob %v", blob.Name)
		err = blob.Delete(nil)
		if err != nil {
			return errors.Wrapf(err, "error deleting blob %v : %v", blob.Name, err)
		}
	}
	return nil
}

func (b *etcdBackup) Delete(name string) error {
	blob := b.etcdContainer.GetBlobReference(name)
	return blob.Delete(nil)
}

func (b *etcdBackup) Retrieve(srcBlob, destPath string) error {
	b.log.Printf("copy blob %v to filesystem %v", srcBlob, destPath)
	blob := b.etcdContainer.GetBlobReference(srcBlob)
	var rc io.ReadCloser
	var err error
	for {
		rc, err = blob.Get(&azstorage.GetBlobOptions{})
		if err, ok := err.(*url.Error); ok && err.Timeout() {
			time.Sleep(time.Second)
			fmt.Println("timeout: retrying")
			continue
		}
		break
	}
	if err != nil {
		return err
	}
	defer rc.Close()

	df, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer df.Close()

	_, err = io.Copy(df, rc)
	return err
}
