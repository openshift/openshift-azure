package storage

import (
	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

// FakeStorageClient is a mock of StorageClient interface
type FakeStorageClient struct {
	rp *StorageRP
}

// NewFakeStorageClient creates a new mock instance
func NewFakeStorageClient(rp *StorageRP) *FakeStorageClient {
	return &FakeStorageClient{rp: rp}
}

func (s *FakeStorageClient) GetBlobService() storage.BlobStorageClient {
	return &FakeBlobStorageClient{rp: s.rp}
}
