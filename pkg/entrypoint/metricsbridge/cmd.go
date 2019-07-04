package metricsbridge

import (
	"github.com/spf13/cobra"

	"github.com/openshift/openshift-azure/pkg/entrypoint/config"
)

type cmdConfig struct {
	config.Common
	configDir       string
	httpPort        int
	metricsEndpoint string
}

// NewCommand returns the cobra command for "metricsbridge".
func NewCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:  "metricsbridge",
		Long: "Start metrics-bridge application",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configFromCmd(cmd)
			if err != nil {
				return err
			}
			return start(cfg)
		},
	}
	cc.Flags().String("config", "", "config file location")
	cc.Flags().Int("http-port", 8080, "The http server port")
	cc.Flags().String("metrics-endpoint", "/metrics", "The endpoint for serving metricsbridge metrics")
	cobra.MarkFlagRequired(cc.Flags(), "config")

	return cc
}

func configFromCmd(cmd *cobra.Command) (*cmdConfig, error) {
	c := &cmdConfig{}
	var err error
	c.Common, err = config.CommonConfigFromCmd(cmd)
	if err != nil {
		return nil, err
	}
	c.configDir, err = cmd.Flags().GetString("config")
	if err != nil {
		return nil, err
	}
	c.httpPort, err = cmd.Flags().GetInt("http-port")
	if err != nil {
		return nil, err
	}
	c.metricsEndpoint, err = cmd.Flags().GetString("metrics-endpoint")
	if err != nil {
		return nil, err
	}
	return c, nil
}
