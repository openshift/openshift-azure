package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
	"github.com/openshift/openshift-azure/pkg/util/roundtrippers"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

type conn struct {
	net.Conn
	r *bufio.Reader
}

func (c *conn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

type Server struct {
	log        *logrus.Entry
	testConfig api.TestConfig
}

func (s *Server) configureProxyDialer(cs *api.OpenShiftManagedCluster) error {
	proxyEnvName := fmt.Sprintf("PROXYURL_%s", strings.ToUpper(cs.Location))
	if s.testConfig.RunningUnderTest && s.testConfig.ProxyURL == "" {
		s.testConfig.ProxyURL = os.Getenv(proxyEnvName)
	}
	s.log.Debugf("%s is %s", proxyEnvName, s.testConfig.ProxyURL)

	roots := x509.NewCertPool()
	if ok := roots.AppendCertsFromPEM(s.testConfig.ProxyCa); !ok {
		return fmt.Errorf("error configuring proxy")
	}

	roundtrippers.DialHook = func(network, address string) (net.Conn, error) {
		s.log.Debugf("dial %s", address)
		/* #nosec - connecting to external IP of a FakeRP cluster, expect self signed cert */
		c, err := tls.Dial("tcp", s.testConfig.ProxyURL, &tls.Config{
			RootCAs:      roots,
			Certificates: []tls.Certificate{s.testConfig.ProxyCertificate},
			// TOFIX: Current certificate does not contain
			// SANs/IPs. This causes validation error. Need to regenerate
			// new certificate and remove this
			InsecureSkipVerify: true,
		})
		if err != nil {
			s.log.Error(err)
			return nil, err
		}

		r := bufio.NewReader(c)

		req, err := http.NewRequest(http.MethodConnect, "", nil)
		if err != nil {
			return nil, err
		}
		req.Host = address

		err = req.Write(c)
		if err != nil {
			return nil, err
		}

		resp, err := http.ReadResponse(r, req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
		}

		return &conn{Conn: c, r: r}, nil
	}
	return nil
}

func (s *Server) healthz(cs *api.OpenShiftManagedCluster) error {
	pool := x509.NewCertPool()
	pool.AddCert(cs.Config.Certificates.Ca.Cert)
	tlsConfig := tls.Config{
		RootCAs:    pool,
		ServerName: cs.Properties.FQDN,
	}

	client := &http.Client{
		Transport: roundtrippers.HealthCheck(cs.Properties.FQDN, cs.Location, cs.Properties.NetworkProfile.PrivateEndpoint, s.testConfig, &tlsConfig),
		Timeout:   10 * time.Second,
	}

	//url := "https://" + cs.Properties.FQDN + "/healthz"
	url := "https://" + *cs.Properties.NetworkProfile.PrivateEndpoint + "/healthz"

	_, err := wait.ForHTTPStatusOk(context.Background(), s.log, client, url, time.Second)
	return err
}

func (s *Server) console(cs *api.OpenShiftManagedCluster) error {
	cert := cs.Config.Certificates.OpenShiftConsole.Certs
	pool := x509.NewCertPool()
	pool.AddCert(cert[len(cert)-1])
	tlsConfig := tls.Config{
		RootCAs:    pool,
		ServerName: cs.Properties.PublicHostname,
	}

	client := &http.Client{
		Transport: roundtrippers.HealthCheck(cs.Properties.FQDN, cs.Location, cs.Properties.NetworkProfile.PrivateEndpoint, s.testConfig, &tlsConfig),
		Timeout:   10 * time.Second,
	}

	url := "https://" + *cs.Properties.NetworkProfile.PrivateEndpoint + "/console/"

	_, err := wait.ForHTTPStatusOk(context.Background(), s.log, client, url, time.Second)
	return err
}

func (s *Server) kubeclient(cs *api.OpenShiftManagedCluster, cert *tls.Certificate) error {
	restconfig, err := managedcluster.RestConfigFromV1Config(cs.Config.AdminKubeconfig)
	if err != nil {
		return err
	}
	tlsConfig, err := restclient.TLSConfigFor(restconfig)
	if err != nil {
		return err
	}
	restconfig.Host = *cs.Properties.NetworkProfile.PrivateEndpoint
	restconfig.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		return &http.Transport{
			Proxy: func(*http.Request) (*url.URL, error) {
				return url.Parse(fmt.Sprintf("https://%s:8443/", os.Getenv(fmt.Sprintf("PROXYURL_%s", strings.ToUpper(cs.Location)))))
			},
			TLSClientConfig:     tlsConfig,
			TLSHandshakeTimeout: 10 * time.Second,
		}
	}
	s.log.Infoln("get clients")
	cli, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		return err
	}
	s.log.Infoln("get pods")
	pods, err := cli.CoreV1().Pods("kube-system").List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		s.log.Infoln(pod.Name)
	}
	return err
}

func main() {
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	logger.SetLevel(logrus.DebugLevel)
	log := logrus.NewEntry(logger)

	b, err := ioutil.ReadFile("./_data/containerservice.yaml")
	if err != nil {
		panic(err)
	}

	var cs *api.OpenShiftManagedCluster
	err = yaml.Unmarshal(b, &cs)
	if err != nil {
		panic(err)
	}
	cs.Properties.NetworkProfile.PrivateEndpoint = to.StringPtr("172.30.1.6")

	s := Server{log: log}
	s.testConfig = fakerp.GetTestConfig()

	s.configureProxyDialer(cs)

	err = s.healthz(cs)
	if err == nil {
		log.Infoln("healthz: success!")
	} else {
		log.Errorf("healthz: %s", err)
	}
	err = s.console(cs)
	if err == nil {
		log.Infoln("console: success!")
	} else {
		log.Errorf("console: %s", err)
	}
	//err = s.kubeclient(cs)
}
