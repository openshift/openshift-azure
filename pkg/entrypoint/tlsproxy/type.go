package tlsproxy

import (
	"net/http"
	"net/url"
	"regexp"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/entrypoint/config"
)

type Config struct {
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
