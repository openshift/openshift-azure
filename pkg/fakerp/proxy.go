package fakerp

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/roundtrippers"
)

type conn struct {
	net.Conn
	r *bufio.Reader
}

func (c *conn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

func (s *Server) configureProxyDialer(cs *api.OpenShiftManagedCluster) error {
	if s.testConfig.RunningUnderTest && s.testConfig.ProxyURL == "" {
		proxyEnvName := fmt.Sprintf("PROXYURL_%s", strings.ToUpper(cs.Location))
		s.testConfig.ProxyURL = os.Getenv(proxyEnvName)
		s.log.Debugf("%s is %s", proxyEnvName, s.testConfig.ProxyURL)
	}

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
