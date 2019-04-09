package sync

import (
	"time"

	"github.com/openshift/openshift-azure/pkg/entrypoint/config"
)

type Config struct {
	config.Common
	dryRun   bool
	once     bool
	interval time.Duration
}
