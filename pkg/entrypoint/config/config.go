package config

import (
	"github.com/spf13/cobra"
)

type Common struct {
	LogLevel string
}

func CommonConfigFromCmd(cmd *cobra.Command) (Common, error) {
	logLevel, err := cmd.Flags().GetString("loglevel")
	if err != nil {
		return Common{}, err
	}
	return Common{
		LogLevel: logLevel,
	}, nil
}
