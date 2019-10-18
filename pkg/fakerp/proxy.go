package fakerp

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/openshift/openshift-azure/pkg/util/roundtrippers"
)

type conn struct {
	net.Conn
	r *bufio.Reader
}

func (c *conn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

// ConfigureProxyDialer also called by e2e tests
func ConfigureProxyDialer() error {
	// load proxy configuration for tests
	var cert tls.Certificate
	roots := x509.NewCertPool()

	secretsPath := "secrets"
	if _, err := os.Stat(path.Join(secretsPath, "proxy-client.pem")); os.IsNotExist(err) {
		secretsPath = "../../secrets"
	}
	cert, err := tls.LoadX509KeyPair(path.Join(secretsPath, "proxy-client.pem"), path.Join(secretsPath, "proxy-client.key"))
	if err != nil {
		return err
	}
	ca, err := ioutil.ReadFile(path.Join(secretsPath, "proxy-ca.pem"))
	if err != nil {
		return err
	}
	if ok := roots.AppendCertsFromPEM(ca); !ok {
		return fmt.Errorf("error configuring proxy")
	}

	roundtrippers.PrivateEndpointDialHook = func(location string) func(network, address string) (net.Conn, error) {
		return func(network, address string) (net.Conn, error) {
			proxyEnvName := "PROXYURL_" + strings.ToUpper(location)
			proxyURL := os.Getenv(proxyEnvName)
			if proxyURL == "" {
				return nil, fmt.Errorf("%s not set", proxyEnvName)
			}

			c, err := tls.Dial("tcp", proxyURL, &tls.Config{
				RootCAs:      roots,
				Certificates: []tls.Certificate{cert},
				ServerName:   "proxy-server",
			})
			if err != nil {
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
	}

	return nil
}
