package etcdbackup

import (
	"github.com/openshift/openshift-azure/pkg/entrypoint/config"
)

type Config struct {
	config.Common
	BlobName    string
	Destination string
	MaxBackups  int
	Action      string
}
