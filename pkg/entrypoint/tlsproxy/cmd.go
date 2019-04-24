package tlsproxy

import (
	"net/http"
	"net/url"
	"regexp"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/openshift/openshift-azure/pkg/entrypoint/config"
)

type cmdConfig struct {
	config.Common
	listen      string
	insecure    bool
	caCert      string
	cert        string
	key         string
	servingCert string
	servingKey  string
	whitelist   string
	hostname    string

	username string
	password string
	// transformed fields
	log             *logrus.Entry
	cli             *http.Client
	redirectURL     *url.URL
	whitelistRegexp *regexp.Regexp
}

// NewCommand returns the cobra command for "tlsproxy".
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

func configFromCmd(cmd *cobra.Command) (*cmdConfig, error) {
	c := &cmdConfig{}
	var err error
	c.Common, err = config.CommonConfigFromCmd(cmd)
	if err != nil {
		return nil, err
	}
	c.listen, err = cmd.Flags().GetString("listen")
	if err != nil {
		return nil, err
	}
	c.insecure, err = cmd.Flags().GetBool("insecure")
	if err != nil {
		return nil, err
	}
	c.caCert, err = cmd.Flags().GetString("cacert")
	if err != nil {
		return nil, err
	}
	c.cert, err = cmd.Flags().GetString("cert")
	if err != nil {
		return nil, err
	}
	c.key, err = cmd.Flags().GetString("key")
	if err != nil {
		return nil, err
	}
	c.servingKey, err = cmd.Flags().GetString("servingkey")
	if err != nil {
		return nil, err
	}
	c.servingCert, err = cmd.Flags().GetString("servingcert")
	if err != nil {
		return nil, err
	}
	c.whitelist, err = cmd.Flags().GetString("whitelist")
	if err != nil {
		return nil, err
	}
	c.hostname, err = cmd.Flags().GetString("hostname")
	if err != nil {
		return nil, err
	}
	return c, nil
}
