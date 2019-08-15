package tlsproxy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"github.com/openshift/openshift-azure/pkg/util/log"
)

func (c *cmdConfig) validate() (errs []error) {
	if _, err := url.Parse(c.hostname); err != nil {
		errs = append(errs, fmt.Errorf("invalid hostname %q", c.hostname))
	}
	if c.password == "" && c.username != "" ||
		c.password != "" && c.username == "" {
		errs = append(errs, fmt.Errorf("if either USERNAME or PASSWORD environment variable is unset, both must be unset"))
	}

	return
}

func (c *cmdConfig) Init() error {
	logrus.SetLevel(log.SanitizeLogLevel(c.LogLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	c.log = logrus.NewEntry(logrus.StandardLogger())

	c.username = os.Getenv("USERNAME")
	c.password = os.Getenv("PASSWORD")

	// validate flags
	if errs := c.validate(); len(errs) > 0 {
		return errors.Wrap(kerrors.NewAggregate(errs), "validation failed")
	}

	// sanitize inputs
	var err error
	c.redirectURL, err = url.Parse(c.hostname)
	if err != nil {
		return err
	}

	if c.whitelist != "" {
		c.whitelistRegexp, err = regexp.Compile(c.whitelist)
		if err != nil {
			return err
		}
	}

	cert, err := tls.LoadX509KeyPair(c.cert, c.key)
	if err != nil {
		return err
	}

	tlsClientConfig := &tls.Config{
		InsecureSkipVerify: c.insecure,
		Certificates:       []tls.Certificate{cert},
	}

	if c.caCert != "" {
		b, err := ioutil.ReadFile(c.caCert)
		if err != nil {
			return err
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(b)
		tlsClientConfig.RootCAs = pool
	}

	c.cli = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsClientConfig,
		},
	}

	return nil
}

func (c *cmdConfig) Run() error {
	handlerFunc := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		req.URL.Scheme = c.redirectURL.Scheme
		req.URL.Host = c.redirectURL.Host
		req.RequestURI = ""
		req.Host = ""

		if c.whitelist != "" {
			if !c.whitelistRegexp.MatchString(req.URL.Path) || req.Method != http.MethodGet {
				http.Error(rw, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
		}

		// check authentication
		if c.username != "" {
			if !c.authIsOK(rw, req) {
				http.Error(rw, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
		}

		resp, err := c.cli.Do(req)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		for k, v := range resp.Header {
			rw.Header()[k] = v
		}
		rw.WriteHeader(resp.StatusCode)
		io.Copy(rw, resp.Body)
	})

	if c.servingCert != "" && c.servingKey != "" {
		c.log.Debug("starting in reencrypt mode")
		return http.ListenAndServeTLS(c.listen, c.servingCert, c.servingKey, handlerFunc)
	}
	c.log.Debug("starting in plain text mode")
	return http.ListenAndServe(c.listen, handlerFunc)
}

func (c *cmdConfig) authIsOK(rw http.ResponseWriter, req *http.Request) bool {
	username, password, _ := req.BasicAuth()
	return username == c.username && password == c.password
}

func start(cfg *cmdConfig) error {
	logrus.SetLevel(log.SanitizeLogLevel(cfg.LogLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log := logrus.NewEntry(logrus.StandardLogger())

	cfg.log = log

	err := cfg.Init()
	if err != nil {
		return err
	}

	cfg.log.Infof("tlsproxy starting at %s", cfg.listen)
	return cfg.Run()
}
