package storage

import (
	"bytes"
	"io"
	"io/ioutil"
	"syscall"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/errors"
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
	log *logrus.Entry
}

var _ Client = &client{}

// NewClient returns a new Client
func NewClient(log *logrus.Entry, accountName, accountKey, serviceBaseURL, apiVersion string, useHTTPS bool) (Client, error) {
	azs, err := storage.NewClient(accountName, accountKey, serviceBaseURL, apiVersion, useHTTPS)
	if err != nil {
		return nil, err
	}

	return &client{
		Client: azs,
		log:    log,
	}, nil
}

func (c *client) GetBlobService() BlobStorageClient {
	return &blobStorageClient{
		BlobStorageClient: c.Client.GetBlobService(),
		log:               c.log,
	}
}

// BlobStorageClient is a minimal interface for azure BlobStorageClient
type BlobStorageClient interface {
	GetContainerReference(name string) Container
}

type blobStorageClient struct {
	storage.BlobStorageClient
	log *logrus.Entry
}

var _ BlobStorageClient = &blobStorageClient{}

func (c *blobStorageClient) GetContainerReference(name string) Container {
	return &container{
		Container: c.BlobStorageClient.GetContainerReference(name),
		log:       c.log,
	}
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
	log *logrus.Entry
}

var _ Container = &container{}

func (c *container) GetBlobReference(name string) Blob {
	return &blob{
		Blob: c.Container.GetBlobReference(name),
		log:  c.log,
	}
}

// Blob is a minimal interface for azure Blob
type Blob interface {
	CreateBlockBlobFromReader(blob io.Reader, options *storage.PutBlobOptions) error
	CreateBlockBlob(options *storage.PutBlobOptions) error
	PutBlock(blockID string, chunk []byte, options *storage.PutBlockOptions) error
	PutBlockList(blocks []storage.Block, options *storage.PutBlockListOptions) error
	Get(options *storage.GetBlobOptions) (io.ReadCloser, error)
	GetSASURI(options storage.BlobSASOptions) (string, error)
	Delete(options *storage.DeleteBlobOptions) error
	Exists() (bool, error)
}

type blob struct {
	*storage.Blob
	log *logrus.Entry
}

var _ Blob = &blob{}

func (b *blob) CreateBlockBlobFromReader(blob io.Reader, options *storage.PutBlobOptions) error {
	data, err := ioutil.ReadAll(blob)
	if err != nil {
		return err
	}

	retry, retries := 0, 3
	for {
		retry++

		err := b.Blob.CreateBlockBlobFromReader(bytes.NewReader(data), options)
		if err == nil {
			return nil
		}

		if retry <= retries && errors.IsMatchingSyscallError(err, syscall.ECONNRESET) {
			b.log.Warnf("%s: retry %d", err, retry)
			continue
		}

		return err
	}
}
