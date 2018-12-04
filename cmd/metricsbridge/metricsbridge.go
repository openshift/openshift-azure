package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/log"
	"github.com/openshift/openshift-azure/pkg/util/statsd"
)

var (
	logLevel  = flag.String("loglevel", "Debug", "Valid values are Debug, Info, Warning, Error")
	gitCommit = "unknown"
)

type config struct {
	Interval                   time.Duration `json:"intervalNanoseconds,omitempty"`
	PrometheusFederateEndpoint string        `json:"prometheusFederateEndpoint,omitempty"`
	StatsdEndpoint             string        `json:"statsdEndpoint,omitempty"`

	Series []string `json:"series,omitempty"`

	Namespace string `json:"namespace,omitempty"`
	Region    string `json:"region,omitempty"`

	Token              string `json:"token,omitempty"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty"`

	log     *logrus.Entry
	rootCAs *x509.CertPool
	http    *http.Client
	statsd  *statsd.Client

	ready bool
}

func (c *config) load(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(b, &c); err != nil {
		return err
	}

	b, err = ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt")
	switch {
	case os.IsNotExist(err):
	case err != nil:
		return err
	default:
		c.rootCAs = x509.NewCertPool()
		c.rootCAs.AppendCertsFromPEM(b)
	}

	b, err = ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	switch {
	case os.IsNotExist(err):
	case err != nil:
		return err
	default:
		c.Token = string(b)
	}

	return nil
}

func (c *config) validate() (errs []error) {
	if c.Interval < time.Second {
		errs = append(errs, fmt.Errorf("intervalNanoseconds %q too small", c.Interval))
	}
	if _, err := url.Parse(c.PrometheusFederateEndpoint); err != nil {
		errs = append(errs, fmt.Errorf("prometheusFederateEndpoint: %s", err))
	}
	if _, err := net.ResolveUDPAddr("udp", c.StatsdEndpoint); err != nil {
		errs = append(errs, fmt.Errorf("statsdEndpoint: %s", err))
	}
	if len(c.Series) == 0 {
		errs = append(errs, fmt.Errorf("must configure at least one series"))
	}

	return
}

func (c *config) init() error {
	var err error

	c.statsd, err = statsd.NewClient(c.log, c.StatsdEndpoint)
	if err != nil {
		return err
	}

	c.http = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:            c.rootCAs,
				InsecureSkipVerify: c.InsecureSkipVerify,
			},
		},
	}
	// not ready by default
	c.ready = false
	go c.health()
	return nil
}

func run(log *logrus.Entry, configpath string) error {
	c := &config{log: log}

	if err := c.load(configpath); err != nil {
		return err
	}

	if errs := c.validate(); len(errs) > 0 {
		var sb strings.Builder
		for _, err := range errs {
			sb.WriteString(err.Error())
			sb.WriteByte('\n')
		}
		return errors.New(sb.String())
	}

	if err := c.init(); err != nil {
		return err
	}

	return c.run()
}

func (c *config) run() error {
	prometheusURL, err := url.Parse(c.PrometheusFederateEndpoint)
	if err != nil {
		return err
	}

	v := url.Values{}
	for _, s := range c.Series {
		v.Add("match[]", s)
	}
	prometheusURL.RawQuery = v.Encode()

	req, err := http.NewRequest(http.MethodGet, prometheusURL.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "Bearer "+c.Token)
	req.Header.Add("Accept", string(expfmt.FmtText))

	t := time.NewTicker(c.Interval)
	defer t.Stop()

	for {
		if err := c.runOnce(req); err != nil {
			c.log.Warn(err)
		}
		<-t.C
	}
}

func (c *config) runOnce(req *http.Request) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.ready = false
		return fmt.Errorf("prometheus returned status code %d", resp.StatusCode)
	}
	c.ready = true
	d := expfmt.NewDecoder(resp.Body, expfmt.FmtText)

	for {
		var family io_prometheus_client.MetricFamily

		err = d.Decode(&family)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		for _, m := range family.Metric {
			gauge := &statsd.Gauge{
				Namespace: c.Namespace,
				Metric:    *family.Name,
				Dims: map[string]string{
					"Region":       c.Region,
					"UnderlayName": "",
				},
				Value: *m.Untyped.Value,
			}
			for _, label := range m.Label {
				gauge.Dims[*label.Name] = *label.Value
			}
			if err = c.statsd.Write(gauge); err != nil {
				return err
			}
		}
	}

	return c.statsd.Flush()
}

func (c *config) health() {
	c.log.Debug("starting health endpoints")
	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	},
	)
	http.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
		if c.ready {
			// 200
			w.WriteHeader(http.StatusOK)
		} else {
			// 503
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	},
	)
	http.ListenAndServe(":8080", nil)
}

func main() {
	flag.Parse()
	logrus.SetLevel(log.SanitizeLogLevel(*logLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	log := logrus.NewEntry(logrus.StandardLogger())
	log.Printf("metricsbridge starting, git commit %s", gitCommit)

	if len(os.Args) != 2 {
		log.Fatalf("usage: %s config.yaml", os.Args[0])
	}

	if err := run(log, os.Args[1]); err != nil {
		log.Fatal(err)
	}
}
