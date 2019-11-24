package admissioncontroller

import (
	"github.com/spf13/cobra"

	"github.com/openshift/openshift-azure/pkg/entrypoint/config"
)

type cmdConfig struct {
	config.Common
	configFile string
}

// NewCommand returns the cobra command for "admissioncontroller".
func NewCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:  "admissioncontroller",
		Long: "Start ARO Admission Controller",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configFromCmd(cmd)
			if err != nil {
				return err
			}
			return start(cfg)
		},
	}
	cc.Flags().String("configfile", "/etc/aro-admission-controller/aro-admission-controller.yaml", "configuration file for ARO admission controller")

	return cc
}

func configFromCmd(cmd *cobra.Command) (*cmdConfig, error) {
	c := &cmdConfig{}
	var err error
	c.Common, err = config.CommonConfigFromCmd(cmd)
	if err != nil {
		return nil, err
	}
	c.configFile, err = cmd.Flags().GetString("configfile")
	if err != nil {
		return nil, err
	}

	return c, nil
}
