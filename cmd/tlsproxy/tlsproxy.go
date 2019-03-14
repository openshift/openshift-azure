package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
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

var (
	logLevel    = flag.String("loglevel", "Info", "Valid values are Debug, Info, Warning, Error")
	listen      = flag.String("listen", ":8080", "IP/port to listen on")
	insecure    = flag.Bool("insecure", false, "don't validate CA certificate")
	cacert      = flag.String("cacert", "", "file containing CA certificate(s) for the rewrite hostname")
	cert        = flag.String("cert", "", "file containing client certificate for the rewrite hostname")
	key         = flag.String("key", "", "file containing client key for the rewrite hostname")
	servingkey  = flag.String("servingkey", "", "file containing serving key for re-encryption")
	servingcert = flag.String("servingcert", "", "file containing serving certificate for re-encryption")
	whitelist   = flag.String("whitelist", "", "URL whitelist regular expression")
	hostname    = flag.String("hostname", "", "Hostname value to rewrite. Example: https://hostname.to.rewrite/")
	gitCommit   = "unknown"
)

type config struct {
	username string
	password string
	// transformed fields
	log             *logrus.Entry
	cli             *http.Client
	redirectURL     *url.URL
	whitelistRegexp *regexp.Regexp
}

func (c *config) validate() (errs []error) {
	if _, err := url.Parse(*hostname); err != nil {
		errs = append(errs, fmt.Errorf("invalid hostname %q", *hostname))
	}
	if c.password == "" && c.username != "" ||
		c.password != "" && c.username == "" {
		errs = append(errs, fmt.Errorf("if either USERNAME or PASSWORD environment variable is set, both must be set"))
	}

	return
}

func (c *config) Init() error {
	logrus.SetLevel(log.SanitizeLogLevel(*logLevel))
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
	c.redirectURL, err = url.Parse(*hostname)
	if err != nil {
		return err
	}

	c.whitelistRegexp, err = regexp.Compile(*whitelist)
	if err != nil {
		return err
	}

	cert, err := tls.LoadX509KeyPair(*cert, *key)
	if err != nil {
		return err
	}

	tlsClientConfig := &tls.Config{
		InsecureSkipVerify: *insecure,
		Certificates:       []tls.Certificate{cert},
	}

	if *cacert != "" {
		b, err := ioutil.ReadFile(*cacert)
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

func (c *config) Run() error {
	handlerFunc := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		req.URL.Scheme = c.redirectURL.Scheme
		req.URL.Host = c.redirectURL.Host
		req.RequestURI = ""
		req.Host = ""

		if !c.whitelistRegexp.MatchString(req.URL.String()) || req.Method != http.MethodGet {
			http.Error(rw, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
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

	if *servingcert != "" && *servingkey != "" {
		c.log.Debug("starting in reencrypt mode")
		return http.ListenAndServeTLS(*listen, *servingcert, *servingkey, handlerFunc)
	}
	c.log.Debug("starting in plain text mode")
	return http.ListenAndServe(*listen, handlerFunc)
}

func (c *config) authIsOK(rw http.ResponseWriter, req *http.Request) bool {
	username, password, _ := req.BasicAuth()
	return username == c.username && password == c.password
}

func main() {
	flag.Parse()

	c := config{}
	err := c.Init()
	if err != nil {
		panic(err)
	}

	c.log.Infof("tlsproxy starting at %s, git commit %s", *listen, gitCommit)
	if err := c.Run(); err != nil {
		c.log.Fatal(err)
	}
}
