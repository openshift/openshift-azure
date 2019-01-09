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
	"syscall"
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

	omitLabels = map[string]struct{}{ // these must be lower case
		"cluster":            {},
		"prometheus":         {},
		"prometheus_replica": {},

		// we populate these
		"region":            {},
		"subscriptionid":    {},
		"resourcegroupname": {},
	}
)

type config struct {
	Interval                   time.Duration `json:"intervalNanoseconds,omitempty"`
	PrometheusFederateEndpoint string        `json:"prometheusFederateEndpoint,omitempty"`
	StatsdSocket               string        `json:"statsdSocket,omitempty"`

	Series []string `json:"series,omitempty"`

	Account   string `json:"account,omitempty"`
	Namespace string `json:"namespace,omitempty"`

	Region            string `json:"region,omitempty"`
	SubscriptionID    string `json:"subscriptionId,omitempty"`
	ResourceGroupName string `json:"resourceGroupName,omitempty"`

	Token              string `json:"token,omitempty"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty"`

	log     *logrus.Entry
	rootCAs *x509.CertPool
	http    *http.Client
	conn    net.Conn
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

func (c *config) defaultAndValidate() (errs []error) {
	if c.Interval == 0 {
		c.Interval = time.Minute
	}

	if c.Interval < time.Second {
		errs = append(errs, fmt.Errorf("intervalNanoseconds %q too small", int64(c.Interval)))
	}
	if _, err := url.Parse(c.PrometheusFederateEndpoint); err != nil {
		errs = append(errs, fmt.Errorf("prometheusFederateEndpoint: %s", err))
	}
	if _, err := net.ResolveUnixAddr("unix", c.StatsdSocket); err != nil {
		errs = append(errs, fmt.Errorf("statsdSocket: %s", err))
	}
	if len(c.Series) == 0 {
		errs = append(errs, fmt.Errorf("must configure at least one series"))
	}

	return
}

func (c *config) init() error {
	for {
		var err error
		c.conn, err = net.Dial("unix", c.StatsdSocket)
		if err == nil {
			break
		}
		if err, ok := err.(*net.OpError); ok {
			if err, ok := err.Err.(*os.SyscallError); ok {
				if err.Err == syscall.ENOENT {
					c.log.Warn("socket not found, sleeping...")
					time.Sleep(5 * time.Second)
					continue
				}
			}
		}
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

	return nil
}

func run(log *logrus.Entry, configpath string) error {
	c := &config{log: log}

	if err := c.load(configpath); err != nil {
		return err
	}

	if errs := c.defaultAndValidate(); len(errs) > 0 {
		var sb strings.Builder
		for _, err := range errs {
			sb.WriteString(err.Error())
			sb.WriteByte('\n')
		}
		return errors.New(sb.String())
	}

	if c.Interval != time.Minute {
		log.Warnf("intervalNanoseconds is set to %q.  It must be set to %q in production", int64(c.Interval), int64(time.Minute))
	}

	if err := c.init(); err != nil {
		return err
	}
	defer c.conn.Close()

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
	now := time.Now()

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("prometheus returned status code %d", resp.StatusCode)
	}

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
			f := &statsd.Float{
				Metric:    *family.Name,
				Account:   c.Account,
				Namespace: c.Namespace,
				Dims:      map[string]string{},
				TS:        now,
				Value:     *m.Untyped.Value,
			}
			for _, label := range m.Label {
				if _, found := omitLabels[strings.ToLower(*label.Name)]; found {
					continue
				}
				f.Dims[*label.Name] = *label.Value
			}
			if c.Region != "" {
				f.Dims["region"] = c.Region
			}
			if c.SubscriptionID != "" {
				f.Dims["subscriptionId"] = c.SubscriptionID
			}
			if c.ResourceGroupName != "" {
				f.Dims["resourceGroupName"] = c.ResourceGroupName
			}
			b, err := f.Marshal()
			if err != nil {
				return err
			}
			if _, err = c.conn.Write(b); err != nil {
				return err
			}
		}
	}

	return nil
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
