package storage

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/Azure/azure-sdk-for-go/storage"
)

type FakeBlobClient struct {
	rp        *StorageRP
	container *FakeContainerClient
	name      string
}

// NewFakeBlobClient creates a new mock instance
func NewFakeBlobClient(rp *StorageRP) *FakeBlobClient {
	return &FakeBlobClient{rp: rp}
}

func (b *FakeBlobClient) CreateBlockBlobFromReader(blob io.Reader, options *storage.PutBlobOptions) error {
	b.rp.Calls = append(b.rp.Calls, "BlobClient:CreateBlockBlobFromReader:"+b.name)
	buf, err := ioutil.ReadAll(blob)
	if err != nil {
		return err
	}
	b.rp.Blobs[b.container.name][b.name] = buf
	return nil
}

func (b *FakeBlobClient) CreateBlockBlob(options *storage.PutBlobOptions) error {
	b.rp.Calls = append(b.rp.Calls, "BlobClient:CreateBlockBlob:"+b.name)
	// nothing to do
	return nil
}

func (b *FakeBlobClient) PutBlock(blockID string, chunk []byte, options *storage.PutBlockOptions) error {
	b.rp.Calls = append(b.rp.Calls, "BlobClient:PutBlock:"+b.name)
	b.rp.Blobs[b.container.name][b.name] = append(b.rp.Blobs[b.container.name][b.name], chunk...)
	return nil
}

func (b *FakeBlobClient) PutBlockList(blocks []storage.Block, options *storage.PutBlockListOptions) error {
	b.rp.Calls = append(b.rp.Calls, "BlobClient:PutBlockList:"+b.name)
	// nothing to do
	return nil
}

func (b *FakeBlobClient) Get(options *storage.GetBlobOptions) (io.ReadCloser, error) {
	b.rp.Calls = append(b.rp.Calls, "BlobClient:Get:"+b.name)
	blob, exist := b.rp.Blobs[b.container.name][b.name]
	if !exist {
		return nil, fmt.Errorf("%s does not exist", b.name)
	}
	return ioutil.NopCloser(bytes.NewReader(blob)), nil
}

func (b *FakeBlobClient) GetSASURI(options storage.BlobSASOptions) (string, error) {
	b.rp.Calls = append(b.rp.Calls, "BlobClient:GetSAURI:"+b.name)
	return fmt.Sprintf("http://example.com/somewhere/%s/%s", b.container.name, b.name), nil
}

func (b *FakeBlobClient) Delete(options *storage.DeleteBlobOptions) error {
	b.rp.Calls = append(b.rp.Calls, "BlobClient:Delete:"+b.name)
	exist, _ := b.Exists()
	if exist {
		delete(b.rp.Blobs[b.container.name], b.name)
	}
	return nil
}

func (b *FakeBlobClient) Exists() (bool, error) {
	_, exist := b.rp.Blobs[b.container.name][b.name]
	return exist, nil
}
