package kubeclient

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	security "github.com/openshift/client-go/security/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	v1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
	"github.com/openshift/openshift-azure/pkg/util/roundtrippers"
)

// NewKubeclient creates a new kubeclient
// If PrivateEndpointIp is not nil - kubeclient roundtripper will point PrivateEndpoint IP address
func NewKubeclient(log *logrus.Entry, config *v1.Config, cs *api.OpenShiftManagedCluster, disableKeepAlives bool, testConfig api.TestConfig) (Interface, error) {
	restconfig, err := NewRestConfig(log, config, cs, disableKeepAlives, testConfig)
	if err != nil {
		return nil, err
	}

	if cs.Properties.NetworkProfile.PrivateEndpoint == nil {
		restconfig.WrapTransport = roundtrippers.NewRetryingRoundTripper(log, disableKeepAlives)
	} else {
		tlsConfig, err := restclient.TLSConfigFor(restconfig)
		if err != nil {
			return nil, err
		}

		restconfig.WrapTransport = roundtrippers.NewPrivateEndpoint(log, cs.Location, *cs.Properties.NetworkProfile.PrivateEndpoint, disableKeepAlives, testConfig, tlsConfig)
		if testConfig.RunningUnderTest {
			//override dialer in manual mode
			restconfig.Host = *cs.Properties.NetworkProfile.PrivateEndpoint
		}
	}
	return newKubeclientFromRestConfig(log, restconfig, disableKeepAlives, testConfig)
}

// NewRestConfig returns restconfig, based on configuration
func NewRestConfig(log *logrus.Entry, config *v1.Config, cs *api.OpenShiftManagedCluster, disableKeepAlives bool, testConfig api.TestConfig) (*rest.Config, error) {
	restconfig, err := managedcluster.RestConfigFromV1Config(config)
	if err != nil {
		return nil, err
	}

	// if we running in PE case - configure RT to use privateEndpoint/proxy
	// newPrivateEndpoint will check if this is runningUnderTest or not
	if cs.Properties.NetworkProfile.PrivateEndpoint != nil {
		log.Debugf("override kubeClient roundtripper with PrivateEndpoint dialer")

		tlsConfig, err := restclient.TLSConfigFor(restconfig)
		if err != nil {
			return nil, err
		}

		restconfig.WrapTransport = newPrivateEndpoint(log, cs, disableKeepAlives, testConfig, tlsConfig)
		if testConfig.RunningUnderTest {
			//override dialer in manual mode
			restconfig.Host = *cs.Properties.NetworkProfile.PrivateEndpoint
		}

	} else {
		restconfig.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
			// first, tweak values on the incoming RoundTripper, which we are
			// relying on being an *http.Transport.

			rt.(*http.Transport).DisableKeepAlives = disableKeepAlives

			// now wrap our RetryingRoundTripper around the incoming RoundTripper.
			return &roundtrippers.RetryingRoundTripper{
				Log:          log,
				RoundTripper: rt,
				Retries:      5,
				GetTimeout:   30 * time.Second,
			}
		}
	}
	return restconfig, nil
}

// newPrivateEndpoint new RoundTripper for private endpoint
func newPrivateEndpoint(log *logrus.Entry, cs *api.OpenShiftManagedCluster, disableKeepAlives bool, testConfig api.TestConfig, tlsConfig *tls.Config) func(rt http.RoundTripper) http.RoundTripper {
	return func(rt http.RoundTripper) http.RoundTripper {
		var rtNew *http.Transport

		// This is development code. This should never ever run in production
		if testConfig.RunningUnderTest {
			tlsConfig.Certificates = append(tlsConfig.Certificates, testConfig.ProxyCertificate)
			tlsConfig.InsecureSkipVerify = true

			// get proxy URL
			// Test settings to use proxy instead of DialTLS
			proxyURL := os.Getenv(fmt.Sprintf("PROXYURL_%s", strings.ToUpper(cs.Location)))

			rtNew = &http.Transport{
				Proxy: func(*http.Request) (*url.URL, error) {
					return url.Parse(fmt.Sprintf("https://%s:8443/", proxyURL))
				},
				TLSClientConfig:     tlsConfig,
				TLSHandshakeTimeout: 10 * time.Second,
			}

			rtNew.DisableKeepAlives = disableKeepAlives

			return &roundtrippers.RetryingRoundTripper{
				Log:          log,
				RoundTripper: rtNew,
				Retries:      5,
				GetTimeout:   30 * time.Second,
			}
		}

		rtNew = &http.Transport{
			DialTLS: func(network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				c, err := net.Dial(network, net.JoinHostPort(*cs.Properties.NetworkProfile.PrivateEndpoint, port))
				if err != nil {
					return nil, err
				}
				tlsConfig.ServerName = host
				return tls.Client(c, tlsConfig), nil
			},
		}

		rtNew.DisableKeepAlives = disableKeepAlives

		// now wrap our RetryingRoundTripper around the incoming RoundTripper.
		return &roundtrippers.RetryingRoundTripper{
			Log:          log,
			RoundTripper: rtNew,
			Retries:      5,
			GetTimeout:   30 * time.Second,
		}
	}
}

// newKubeclient creates a new kubeclient.
// If PrivateEndpointIp is not nil - kubeclient roundtripper
// will dial PrivateEndpoint IP address instead of public API
func newKubeclientFromRestConfig(log *logrus.Entry, restconfig *rest.Config, disableKeepAlives bool, testConfig api.TestConfig) (Interface, error) {
	cli, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	seccli, err := security.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	return &Kubeclientset{
		Log:               log,
		Client:            cli,
		Seccli:            seccli,
		disableKeepAlives: disableKeepAlives,
		restconfig:        restconfig,
		testConfig:        testConfig,
	}, nil
}
