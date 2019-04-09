package azurecontrollers

import (
	"github.com/openshift/openshift-azure/pkg/entrypoint/config"
)

type Config struct {
	config.Common
	dryRun bool
}
