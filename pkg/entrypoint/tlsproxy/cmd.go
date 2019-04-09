package tlsproxy

import (
	"github.com/spf13/cobra"

	"github.com/openshift/openshift-azure/pkg/entrypoint/config"
)

// NewCommand returns the cobra command for "azure-controllers".
func NewCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:  "tlsproxy",
		Long: "Start tlsproxy application",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configFromCmd(cmd)
			if err != nil {
				return err
			}
			return start(cfg)
		},
	}
	cc.Flags().String("listen", ":8080", "IP/port to listen on")
	cc.Flags().Bool("insecure", false, "don't validate CA certificate")
	cc.Flags().String("cacert", "", "file containing CA certificate(s) for the rewrite hostname")
	cc.Flags().String("cert", "", "file containing client certificate for the rewrite hostname")
	cc.Flags().String("key", "", "file containing client key for the rewrite hostname")
	cc.Flags().String("servingkey", "", "file containing serving key for re-encryption")
	cc.Flags().String("servingcert", "", "file containing serving certificate for re-encryption")
	cc.Flags().String("whitelist", "", "URL whitelist regular expression")
	cc.Flags().String("hostname", "", "Hostname value to rewrite. Example: https://hostname.to.rewrite/")
	cobra.MarkFlagRequired(cc.Flags(), "hostname")

	return cc
}

func configFromCmd(cmd *cobra.Command) (*Config, error) {
	c := &Config{}
	var err error
	c.Common, err = config.CommonConfigFromCmd(cmd)
	if err != nil {
		return nil, err
	}
	listen, err := cmd.Flags().GetString("listen")
	if err != nil {
		return nil, err
	}
	insecure, err := cmd.Flags().GetBool("insecure")
	if err != nil {
		return nil, err
	}
	cacert, err := cmd.Flags().GetString("cacert")
	if err != nil {
		return nil, err
	}
	cert, err := cmd.Flags().GetString("cert")
	if err != nil {
		return nil, err
	}
	key, err := cmd.Flags().GetString("key")
	if err != nil {
		return nil, err
	}
	servingkey, err := cmd.Flags().GetString("servingkey")
	if err != nil {
		return nil, err
	}
	servingcert, err := cmd.Flags().GetString("servingcert")
	if err != nil {
		return nil, err
	}
	whitelist, err := cmd.Flags().GetString("whitelist")
	if err != nil {
		return nil, err
	}
	hostname, err := cmd.Flags().GetString("hostname")
	if err != nil {
		return nil, err
	}

	c.listen = listen
	c.insecure = insecure
	c.caCert = cacert
	c.cert = cert
	c.key = key
	c.servingCert = servingcert
	c.servingKey = servingkey
	c.whitelist = whitelist
	c.hostname = hostname

	return c, nil
}
