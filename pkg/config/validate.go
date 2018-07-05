package config

import (
	"fmt"
	"os"

	"github.com/jim-minter/azure-helm/pkg/api"
)

func Validate(m, _ *api.Manifest) error {
	if m.OpenShiftVersion != "3.10" {
		return fmt.Errorf("invalid OpenShiftVersion %q", m.OpenShiftVersion)
	}

	return validateDevelopmentSwitches()
}

func validateDevelopmentSwitches() error {
	switch os.Getenv("DEPLOY_OS") {
	// TODO: when we enable deploying RHEL/Enterprise, uncomment below
	// case "":
	case "centos7":
	default:
		return fmt.Errorf("invalid DEPLOY_OS %q", os.Getenv("DEPLOY_OS"))
	}

	return nil
}
