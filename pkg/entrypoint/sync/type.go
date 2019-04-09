package sync

import (
	"time"

	"github.com/openshift/openshift-azure/pkg/entrypoint/config"
)

type Config struct {
	config.Common
	DryRun   bool
	Once     bool
	Interval time.Duration
}
