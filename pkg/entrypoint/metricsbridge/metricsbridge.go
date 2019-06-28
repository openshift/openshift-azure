package metricsbridge

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/ghodss/yaml"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/cluster/names"
	utilerrors "github.com/openshift/openshift-azure/pkg/util/errors"
	"github.com/openshift/openshift-azure/pkg/util/log"
	"github.com/openshift/openshift-azure/pkg/util/roundtrippers"
	"github.com/openshift/openshift-azure/pkg/util/statsd"
)

/*
curl -Gks \
  -H "Authorization: Bearer $(oc serviceaccounts get-token -n openshift-monitoring prometheus-k8s)" \
  --data-urlencode 'match[]={__name__=~".+"}' \
  https://prometheus-k8s.openshift-monitoring.svc:9091/federate
*/

type authorizingRoundTripper struct {
	http.RoundTripper
	token string
}

func (rt authorizingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+rt.token)
	return rt.RoundTripper.RoundTrip(req)
}

//MetricsConfig stores the configuration of the metricsbridge application
type MetricsConfig struct {
	Interval           time.Duration `json:"intervalNanoseconds,omitempty"`
	PrometheusEndpoint string        `json:"prometheusEndpoint,omitempty"`
	StatsdSocket       string        `json:"statsdSocket,omitempty"`

	Queries []struct {
		Name  string `json:"name,omitempty"`
		Query string `json:"query,omitempty"`
	} `json:"queries,omitempty"`

	Account   string `json:"account,omitempty"`
	Namespace string `json:"namespace,omitempty"`

	Region            string `json:"region,omitempty"`
	SubscriptionID    string `json:"subscriptionId,omitempty"`
	ResourceGroupName string `json:"resourceGroupName,omitempty"`
	ResourceName      string `json:"resourceName,omitempty"`

	Token              string `json:"token,omitempty"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty"`

	log        *logrus.Entry
	rootCAs    *x509.CertPool
	prometheus v1.API
	rt         http.RoundTripper
	conn       net.Conn
}

func (c *MetricsConfig) load(path string) error {
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

func (c *MetricsConfig) defaultAndValidate() (errs []error) {
	if c.Interval == 0 {
		c.Interval = time.Minute
	}

	if c.Interval < time.Second {
		errs = append(errs, fmt.Errorf("intervalNanoseconds %q too small", int64(c.Interval)))
	}
	if _, err := url.Parse(c.PrometheusEndpoint); err != nil {
		errs = append(errs, fmt.Errorf("prometheusEndpoint: %s", err))
	}
	if _, err := net.ResolveUnixAddr("unix", c.StatsdSocket); err != nil {
		errs = append(errs, fmt.Errorf("statsdSocket: %s", err))
	}
	if len(c.Queries) == 0 {
		errs = append(errs, fmt.Errorf("must configure at least one query"))
	}

	return
}

func (c *MetricsConfig) init() error {
	for {
		var err error
		c.log.Debug("dialing statsd socket")
		c.conn, err = net.Dial("unix", c.StatsdSocket)
		if err == nil {
			break
		}
		if utilerrors.IsMatchingSyscallError(err, syscall.ENOENT) {
			c.log.Warn("socket not found, sleeping...")
			time.Sleep(5 * time.Second)
			continue
		}
		return err
	}

	c.log.Debug("dialing prometheus endpoint")
	cli, err := api.NewClient(api.Config{
		Address: c.PrometheusEndpoint,
		RoundTripper: &roundtrippers.AuthorizingRoundTripper{
			RoundTripper: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:            c.rootCAs,
					InsecureSkipVerify: c.InsecureSkipVerify,
				},
			},
			Token: c.Token,
		},
	})
	if err != nil {
		return err
	}

	c.prometheus = v1.NewAPI(cli)

	return nil
}

func run(log *logrus.Entry, configpath string) error {
	c := &MetricsConfig{log: log}

	log.Infof("loading config from %s", configpath)
	if err := c.load(configpath); err != nil {
		return err
	}

	log.Info("validating configuration and adding defaults")
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

	log.Info("initializing...")
	if err := c.init(); err != nil {
		return err
	}
	defer c.conn.Close()

	return c.run()
}

func (c *MetricsConfig) run() error {
	t := time.NewTicker(c.Interval)
	defer t.Stop()

	for {
		if err := c.runOnce(context.Background()); err != nil {
			metricsBridgeErrorsCounter.Inc()
			c.log.Warn(err)
		}
		<-t.C
	}
}

func (c *MetricsConfig) runOnce(ctx context.Context) error {
	startTime := time.Now()
	defer func() {
		metricsBridgeProcessingDurationSummary.Observe(time.Now().Sub(startTime).Seconds())
	}()

	var metricsCount, bytesCount int
	hostnameMap := make(map[string]string)

	c.log.Debug("fetching nodename")
	value, err := c.prometheus.Query(ctx, "node_uname_info", time.Time{})
	if err != nil {
		return err
	}
	for _, nodeSample := range value.(model.Vector) {
		hostnameMap[strings.Split(string(nodeSample.Metric["instance"]), ":")[0]] = string(nodeSample.Metric["nodename"])
	}
	c.log.Debugf("querying %d items", len(c.Queries))
	for _, query := range c.Queries {
		value, err := c.prometheus.Query(ctx, query.Query, time.Time{})
		if err != nil {
			return err
		}

		for _, sample := range value.(model.Vector) {
			f := &statsd.Float{
				Metric:    string(sample.Metric[model.MetricNameLabel]),
				Account:   c.Account,
				Namespace: c.Namespace,
				Dims:      map[string]string{},
				TS:        sample.Timestamp.Time(),
				Value:     float64(sample.Value),
			}
			if query.Name != "" {
				f.Metric = query.Name
			}
			for k, v := range sample.Metric {
				if k != model.MetricNameLabel {
					f.Dims[string(k)] = string(v)
				}
			}
			//if there's an instance dimension try to lookup the hostname
			if instance, found := sample.Metric["instance"]; found {
				hostname, present := hostnameMap[strings.Split(string(instance), ":")[0]]
				if present {
					//only add the hostname dimension if the lookup succeeds
					f.Dims["hostname"] = hostname
					f.Dims["agentrole"] = string(names.GetAgentRole(hostname))
				}
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
			if c.ResourceName != "" {
				f.Dims["resourceName"] = c.ResourceName
			}
			b, err := f.Marshal()
			if err != nil {
				return err
			}
			bytesSent, err := c.conn.Write(b)
			if err != nil {
				return err
			}
			bytesCount += bytesSent
			metricsCount++

			metricsBridgeMetricsTransferredCounter.Inc()
			metricsBridgeBytesTransferredCounter.Add(float64(bytesSent))
		}
	}

	c.log.Infof("sent %d metrics (%d bytes)", metricsCount, bytesCount)

	return nil
}

func start(cfg *cmdConfig) error {
	logrus.SetLevel(log.SanitizeLogLevel(cfg.LogLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log := logrus.NewEntry(logrus.StandardLogger())

	log.Info("starting metrics endpoint")
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.httpPort))
	if err != nil {
		return err
	}

	mux := &http.ServeMux{}
	mux.Handle("/healthz/ready", http.HandlerFunc(readyHandler))
	mux.Handle(cfg.metricsEndpoint, MetricsHandler())

	go http.Serve(l, mux)

	log.Printf("metricsbridge starting")

	if cfg.configDir == "" {
		return fmt.Errorf("config value cant be empty")
	}

	if err := run(log, cfg.configDir); err != nil {
		metricsBridgeErrorsCounter.Inc()
		return err
	}
	return nil
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
