package storage

import (
	"io"

	"github.com/Azure/azure-sdk-for-go/storage"
)

const (
	// DefaultBaseURL is the domain name used for storage requests in the
	// public cloud when a default client is created.
	DefaultBaseURL = storage.DefaultBaseURL

	// DefaultAPIVersion is the Azure Storage API version string used when a
	// basic client is created.
	DefaultAPIVersion = storage.DefaultAPIVersion
)

// Client is a minimal interface for azure Client
type Client interface {
	GetBlobService() BlobStorageClient
}

type client struct {
	storage.Client
}

var _ Client = &client{}

// NewClient returns a new Client
func NewClient(accountName, accountKey, serviceBaseURL, apiVersion string, useHTTPS bool) (Client, error) {
	azs, err := storage.NewClient(accountName, accountKey, serviceBaseURL, apiVersion, useHTTPS)
	if err != nil {
		return nil, err
	}

	return &client{
		Client: azs,
	}, nil
}

func (c *client) GetBlobService() BlobStorageClient {
	return &blobStorageClient{BlobStorageClient: c.Client.GetBlobService()}
}

// BlobStorageClient is a minimal interface for azure BlobStorageClient
type BlobStorageClient interface {
	GetContainerReference(name string) Container
}

type blobStorageClient struct {
	storage.BlobStorageClient
}

var _ BlobStorageClient = &blobStorageClient{}

func (c *blobStorageClient) GetContainerReference(name string) Container {
	return &container{Container: c.BlobStorageClient.GetContainerReference(name)}
}

// Container is a minimal interface for azure Container
type Container interface {
	CreateIfNotExists(options *storage.CreateContainerOptions) (bool, error)
	GetBlobReference(name string) Blob
	Exists() (bool, error)
	ListBlobs(params storage.ListBlobsParameters) (storage.BlobListResponse, error)
}

type container struct {
	*storage.Container
}

var _ Container = &container{}

func (c *container) GetBlobReference(name string) Blob {
	return &blob{Blob: c.Container.GetBlobReference(name)}
}

// Blob is a minimal interface for azure Blob
type Blob interface {
	CreateBlockBlobFromReader(blob io.Reader, options *storage.PutBlobOptions) error
	CreateBlockBlob(options *storage.PutBlobOptions) error
	PutBlock(blockID string, chunk []byte, options *storage.PutBlockOptions) error
	PutBlockList(blocks []storage.Block, options *storage.PutBlockListOptions) error
	Get(options *storage.GetBlobOptions) (io.ReadCloser, error)
	Delete(options *storage.DeleteBlobOptions) error
	Exists() (bool, error)
}

type blob struct {
	*storage.Blob
}

var _ Blob = &blob{}
