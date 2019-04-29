package storage

import (
	azstorage "github.com/Azure/azure-sdk-for-go/storage"

	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

type FakeContainerClient struct {
	rp   *StorageRP
	name string
}

func (c *FakeContainerClient) CreateIfNotExists(options *azstorage.CreateContainerOptions) (bool, error) {
	_, exist := c.rp.Blobs[c.name]
	if !exist {
		c.rp.Blobs[c.name] = map[string][]byte{}
	}
	return !exist, nil
}

func (c *FakeContainerClient) GetBlobReference(name string) storage.Blob {
	return &FakeBlobClient{rp: c.rp, container: c, name: name}
}

func (c *FakeContainerClient) Exists() (bool, error) {
	_, exist := c.rp.Blobs[c.name]
	return exist, nil
}

func (c *FakeContainerClient) ListBlobs(params azstorage.ListBlobsParameters) (azstorage.BlobListResponse, error) {
	bl := azstorage.BlobListResponse{}
	for key := range c.rp.Blobs[c.name] {
		bl.Blobs = append(bl.Blobs, azstorage.Blob{Name: key, Container: &azstorage.Container{Name: c.name}})
	}
	return bl, nil
}
