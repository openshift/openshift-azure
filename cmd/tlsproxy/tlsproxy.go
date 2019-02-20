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
	// Regular expression used to validate RFC1035 hostnames*/
	hostnameRegex = regexp.MustCompile(`^[[:alnum:]][[:alnum:]\-]{0,61}[[:alnum:]]|[[:alpha:]]$`)
)

var (
	logLevel    = flag.String("loglevel", "Debug", "Valid values are Debug, Info, Warning, Error")
	listen      = flag.String("listen", ":8080", "IP/port to listen on")
	insecure    = flag.Bool("insecure", false, "don't validate CA certificate")
	cacert      = flag.String("cacert", "", "file containing CA certificate(s) for the rewrite hostname")
	cert        = flag.String("cert", "", "file containing client certificate for the rewrite hostname")
	key         = flag.String("key", "", "file containing client key for the rewrite hostname")
	reencrypt   = flag.Bool("reencrypt", false, "re-encrypt traffic with other certificate")
	servingkey  = flag.String("servingkey", "", "file containing serving key for re-encryption")
	servingcert = flag.String("servingcert", "", "file containing serving certificate for re-encryption")
	whitelist   = flag.String("whitelist", "", "URL whitelist regular expression")
	hostname    = flag.String("hostname", "", "Hostname value to rewrite. Example: https://hostname.to.rewrite/")
	gitCommit   = "unknown"
)

func validate() []error {
	var errs []error
	if !hostnameRegex.MatchString(*hostname) {
		return append(errs, errors.New("hostname is not a valid hostname"))
	}
	if *cacert == "" {
		return append(errs, errors.New("cacert flag must be provided"))
	}
	if *cert == "" {
		return append(errs, errors.New("cert flag must be provided"))
	}
	if *key == "" {
		return append(errs, errors.New("key flag must be provided"))
	}
	if *reencrypt {
		if *servingkey == "" {
			return append(errs, errors.New("servingkey flag must be provided for re-encrypt"))
		}
		if *servingcert == "" {
			return append(errs, errors.New("servingcert flag must be provided for re-encrypt"))
		}
	}
	return errs
}

func usage() {
	fmt.Printf("Usage:\n")
	fmt.Printf("\"%s -hostname\" url to rewrite \n\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func run(log *logrus.Entry) error {
	redirect, err := url.Parse(*hostname)
	if err != nil {
		return err
	}

	tlsClientConfig := &tls.Config{
		InsecureSkipVerify: *insecure,
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

	if *cert != "" && *key != "" {
		cert, err := tls.LoadX509KeyPair(*cert, *key)
		if err != nil {
			return err
		}
		tlsClientConfig.Certificates = []tls.Certificate{cert}
	}

	cli := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsClientConfig,
		},
	}

	whitelist, err := regexp.Compile(*whitelist)
	if err != nil {
		return err
	}

	handlerFunc := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		req.URL.Scheme = redirect.Scheme
		req.URL.Host = redirect.Host
		req.RequestURI = ""
		req.Host = ""

		if !whitelist.MatchString(req.URL.String()) {
			http.Error(rw, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		log.Debug(req.URL.String())
		resp, err := cli.Do(req)
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

	if *reencrypt {
		log.Debug("starting in reencrypt mode")
		return http.ListenAndServeTLS(*listen, *servingcert, *servingkey, handlerFunc)
	}
	log.Debug("starting in plain text mode")
	return http.ListenAndServe(*listen, handlerFunc)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	logrus.SetLevel(log.SanitizeLogLevel(*logLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log := logrus.NewEntry(logrus.StandardLogger())

	if errs := validate(); len(errs) > 0 {
		log.Error(errors.Wrap(kerrors.NewAggregate(errs), "cannot validate flags"))
		flag.Usage()
	}

	log.Printf("tlsproxy starting, git commit %s", gitCommit)

	if err := run(log); err != nil {
		log.Fatal(err)
	}
}
