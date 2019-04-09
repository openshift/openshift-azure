package metricsbridge

import "github.com/openshift/openshift-azure/pkg/entrypoint/config"

type Config struct {
	config.Common
	ConfigDir string
}
