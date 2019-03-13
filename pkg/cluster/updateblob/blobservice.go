package updateblob

//go:generate go get github.com/golang/mock/gomock
//go:generate go install github.com/golang/mock/mockgen
//go:generate mockgen -destination=../../util/mocks/mock_$GOPACKAGE/blobservice.go -package=mock_$GOPACKAGE -source blobservice.go
//go:generate gofmt -s -l -w ../../util/mocks/mock_$GOPACKAGE/blobservice.go
//go:generate goimports -local=github.com/openshift/openshift-azure -e -w ../../util/mocks/mock_$GOPACKAGE/blobservice.go

import (
	"bytes"
	"encoding/json"

	"github.com/openshift/openshift-azure/pkg/util/azureclient/storage"
)

// here follow well known container and blob names
const (
	UpdateContainerName = "update"
	UpdateBlobName      = "update"
)

type blobService struct {
	updateContainer storage.Container
}

type BlobService interface {
	Read() (*UpdateBlob, error)
	Write(*UpdateBlob) error
}

func NewBlobService(bsc storage.BlobStorageClient) (BlobService, error) {
	u := &blobService{updateContainer: bsc.GetContainerReference(UpdateContainerName)}

	if _, err := u.updateContainer.CreateIfNotExists(nil); err != nil {
		return nil, err
	}

	return u, nil
}

func (u *blobService) Write(blob *UpdateBlob) error {
	data, err := json.Marshal(blob)
	if err != nil {
		return err
	}

	blobRef := u.updateContainer.GetBlobReference(UpdateBlobName)
	return blobRef.CreateBlockBlobFromReader(bytes.NewReader(data), nil)
}

func (u *blobService) Read() (*UpdateBlob, error) {
	blobRef := u.updateContainer.GetBlobReference(UpdateBlobName)
	rc, err := blobRef.Get(nil)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	d := json.NewDecoder(rc)

	var b UpdateBlob
	if err := d.Decode(&b); err != nil {
		return nil, err
	}
	if b.ScalesetHashes == nil {
		b.ScalesetHashes = ScalesetHashes{}
	}
	if b.HostnameHashes == nil {
		b.HostnameHashes = HostnameHashes{}
	}

	return &b, nil
}
