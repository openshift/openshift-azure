package storage

import (
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

type FakeBlobStorageClient struct {
	rp *StorageRP
}

func (bs *FakeBlobStorageClient) GetContainerReference(name string) storage.Container {
	return &FakeContainerClient{rp: bs.rp, name: name}
}
