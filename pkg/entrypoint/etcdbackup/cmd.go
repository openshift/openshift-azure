package etcdbackup

import (
	"github.com/spf13/cobra"

	"github.com/openshift/openshift-azure/pkg/entrypoint/config"
)

// NewCommand returns the cobra command for "etcdbackup".
func NewCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:  "etcdbackup",
		Long: "Start Etcd backup and Restore application",
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configFromCmd(cmd)
			if err != nil {
				return err
			}
			return start(cfg)
		},
	}
	cc.Flags().String("blobname", "", "Name of the blob (without the container)")
	cc.Flags().String("destination", "", "Where to place the blob on the filesystem")
	cc.Flags().Int("maxBackups", 6, "Maximum number of backups to keep")
	cc.Flags().StringP("action", "a", "save", "Action to be executed [save, download]")

	return cc
}

func configFromCmd(cmd *cobra.Command) (*Config, error) {
	c := &Config{}
	var err error
	c.Common, err = config.CommonConfigFromCmd(cmd)
	if err != nil {
		return nil, err
	}
	c.action, err = cmd.Flags().GetString("action")
	if err != nil {
		return nil, err
	}
	c.blobName, err = cmd.Flags().GetString("blobName")
	if err != nil {
		return nil, err
	}
	c.maxBackups, err = cmd.Flags().GetInt("maxBackups")
	if err != nil {
		return nil, err
	}
	c.destination, err = cmd.Flags().GetString("destination")
	if err != nil {
		return nil, err
	}
	return c, nil
}
