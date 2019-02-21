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

	"github.com/openshift/openshift-azure/pkg/api/validate"
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

func (c *config) validate() []error {
	var errs []error
	// check variables
	if *hostname != "" && !validate.IsValidURI(*hostname) {
		return append(errs, errors.New("hostname is not a valid hostname"))
	}
	if !*insecure {
		if *cacert == "" {
			return append(errs, errors.New("cacert must be provided"))
		}
		if *cert == "" {
			return append(errs, errors.New("cert must be provided"))
		}
		if *key == "" {
			return append(errs, errors.New("key must be provided"))
		}
	}
	if *servingkey == "" && *servingcert != "" ||
		*servingkey != "" && *servingcert == "" {
		return append(errs, errors.New("servingkey and servingcert must be provided for re-encrypt"))
	}
	if c.password == "" && c.username != "" ||
		c.password != "" && c.username == "" {
		return append(errs, errors.New("USERNAME and PASSWORD variables must be provided"))
	}

	// check files exist
	if _, err := os.Stat(*cacert); os.IsNotExist(err) {
		return append(errs, errors.New(fmt.Sprintf("file %s does not exist", *cacert)))
	}
	if _, err := os.Stat(*cert); os.IsNotExist(err) {
		return append(errs, errors.New(fmt.Sprintf("file %s does not exist", *cert)))
	}
	if _, err := os.Stat(*key); os.IsNotExist(err) {
		return append(errs, errors.New(fmt.Sprintf("file %s does not exist", *key)))
	}
	if *servingkey != "" && *servingcert != "" {
		if _, err := os.Stat(*servingkey); os.IsNotExist(err) {
			return append(errs, errors.New(fmt.Sprintf("file %s does not exist", *servingkey)))
		}
		if _, err := os.Stat(*servingcert); os.IsNotExist(err) {
			return append(errs, errors.New(fmt.Sprintf("file %s does not exist", *servingcert)))
		}
	}

	return errs
}

func (c *config) Init() error {
	logrus.SetLevel(log.SanitizeLogLevel(*logLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	c.log = logrus.NewEntry(logrus.StandardLogger())

	c.username = os.Getenv("USERNAME")
	c.password = os.Getenv("PASSWORD")

	// validate flags
	if errs := c.validate(); len(errs) > 0 {
		return errors.Wrap(kerrors.NewAggregate(errs), "cannot validate flags")
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

	b, err := ioutil.ReadFile(*cacert)
	if err != nil {
		return err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(b)
	cert, err := tls.LoadX509KeyPair(*cert, *key)
	if err != nil {
		return err
	}

	c.cli = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: *insecure,
				RootCAs:            pool,
				Certificates:       []tls.Certificate{cert},
			},
		},
	}

	return nil
}

func usage() {
	fmt.Printf("Usage:\n")
	fmt.Printf("\"%s -hostname\" url to rewrite \n\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func (c *config) Run() error {
	whitelist, err := regexp.Compile(*whitelist)
	if err != nil {
		return err
	}

	handlerFunc := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		req.URL.Scheme = c.redirectURL.Scheme
		req.URL.Host = c.redirectURL.Host
		req.RequestURI = ""
		req.Host = ""

		if !whitelist.MatchString(req.URL.String()) || req.Method != http.MethodGet {
			http.Error(rw, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		// check authentication
		if c.username != "" {
			if !c.checkAuth(rw, req) {
				http.Error(rw, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
		}

		c.log.Debug(req.URL.String())
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

func (c *config) checkAuth(rw http.ResponseWriter, req *http.Request) bool {
	if username, password, ok := req.BasicAuth(); ok {
		return username == c.username && password == c.password
	}
	return false
}

func main() {
	flag.Usage = usage
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
