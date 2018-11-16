package etcdbackup

import (
	"encoding/base64"
	"io"

	azstorage "github.com/Azure/azure-sdk-for-go/storage"
	"github.com/pborman/uuid"

	azureclientstorage "github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

const (
	// azureBlobBlockChunkLimitInBytes (100MiB is the limit)
	azureBlobBlockChunkLimitInBytes = 100 * 1024 * 1024
)

type blobWriter struct {
	blob   azureclientstorage.Blob
	blocks []azstorage.Block
}

var _ io.WriteCloser = &blobWriter{}

func newBlobWriter(blob azureclientstorage.Blob) (*blobWriter, error) {
	bw := blobWriter{
		blob:   blob,
		blocks: []azstorage.Block{}}
	return &bw, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (w *blobWriter) Write(chunk []byte) (int, error) {
	for i := 0; i < len(chunk); i += azureBlobBlockChunkLimitInBytes {
		j := i + min(len(chunk)-i, azureBlobBlockChunkLimitInBytes)

		id := base64.StdEncoding.EncodeToString([]byte(uuid.New()))

		err := w.blob.PutBlock(id, chunk[i:j], &azstorage.PutBlockOptions{})
		if err != nil {
			return i, err
		}

		w.blocks = append(w.blocks, azstorage.Block{ID: id, Status: azstorage.BlockStatusLatest})
	}

	return len(chunk), nil
}

func (w *blobWriter) Close() error {
	return w.blob.PutBlockList(w.blocks, &azstorage.PutBlockListOptions{})
}
