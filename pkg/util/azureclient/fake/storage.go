package fake

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"

	azurestorage "github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

// FakeStorageClient is a mock of StorageClient interface
type FakeStorageClient struct {
	az *AzureCloud
}

// NewFakeStorageClient creates a new mock instance
func NewFakeStorageClient(az *AzureCloud) *FakeStorageClient {
	return &FakeStorageClient{az: az}
}

// Client mocks base method
func (s *FakeStorageClient) Client() autorest.Client {
	return allwaysDoneClient()
}

func (s *FakeStorageClient) GetBlobService() azurestorage.BlobStorageClient {
	return &FakeBlobStorageClient{az: s.az}
}

type FakeBlobStorageClient struct {
	az *AzureCloud
}

func (bs *FakeBlobStorageClient) GetContainerReference(name string) azurestorage.Container {
	return &FakeContainerClient{az: bs.az, name: name}
}

type FakeContainerClient struct {
	az   *AzureCloud
	name string
}

func (c *FakeContainerClient) CreateIfNotExists(options *storage.CreateContainerOptions) (bool, error) {
	_, exist := c.az.Blobs[c.name]
	if !exist {
		c.az.Blobs[c.name] = map[string][]byte{}
	}
	return !exist, nil
}

func (c *FakeContainerClient) GetBlobReference(name string) azurestorage.Blob {
	return &FakeBlobClient{az: c.az, container: c, name: name}
}

func (c *FakeContainerClient) Exists() (bool, error) {
	_, exist := c.az.Blobs[c.name]
	return exist, nil
}

func (c *FakeContainerClient) ListBlobs(params storage.ListBlobsParameters) (storage.BlobListResponse, error) {
	bl := storage.BlobListResponse{}
	for key := range c.az.Blobs[c.name] {
		bl.Blobs = append(bl.Blobs, storage.Blob{Name: key, Container: &storage.Container{Name: c.name}})
	}
	return bl, nil
}

type FakeBlobClient struct {
	az        *AzureCloud
	container *FakeContainerClient
	name      string
}

// NewFakeStorageClient creates a new mock instance
func NewFakeBlobClient(az *AzureCloud) *FakeBlobClient {
	return &FakeBlobClient{az: az}
}

func (b *FakeBlobClient) CreateBlockBlobFromReader(blob io.Reader, options *storage.PutBlobOptions) error {
	buf, err := ioutil.ReadAll(blob)
	if err != nil {
		return err
	}
	b.az.Blobs[b.container.name][b.name] = buf
	return nil
}

func (b *FakeBlobClient) CreateBlockBlob(options *storage.PutBlobOptions) error {
	// nothing to do
	return nil
}

func (b *FakeBlobClient) PutBlock(blockID string, chunk []byte, options *storage.PutBlockOptions) error {
	b.az.Blobs[b.container.name][b.name] = append(b.az.Blobs[b.container.name][b.name], chunk...)
	return nil
}

func (b *FakeBlobClient) PutBlockList(blocks []storage.Block, options *storage.PutBlockListOptions) error {
	// nothing to do
	return nil
}

func (b *FakeBlobClient) Get(options *storage.GetBlobOptions) (io.ReadCloser, error) {
	blob, exist := b.az.Blobs[b.container.name][b.name]
	if !exist {
		return nil, fmt.Errorf("%s does not exist", b.name)
	}
	return ioutil.NopCloser(bytes.NewReader(blob)), nil
}

func (b *FakeBlobClient) GetSASURI(options storage.BlobSASOptions) (string, error) {
	return fmt.Sprintf("http://example.com/somewhere/%s/%s", b.container.name, b.name), nil
}

func (b *FakeBlobClient) Delete(options *storage.DeleteBlobOptions) error {
	exist, _ := b.Exists()
	if exist {
		delete(b.az.Blobs[b.container.name], b.name)
	}
	return nil
}

func (b *FakeBlobClient) Exists() (bool, error) {
	_, exist := b.az.Blobs[b.container.name][b.name]
	return exist, nil
}
