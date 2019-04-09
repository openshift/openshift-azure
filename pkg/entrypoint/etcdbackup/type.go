package etcdbackup

import (
	"github.com/openshift/openshift-azure/pkg/entrypoint/config"
)

type Config struct {
	config.Common
	blobName    string
	destination string
	maxBackups  int
	action      string
}
