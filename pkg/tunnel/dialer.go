package tunnel

import (
	"crypto/tls"
	"net"
	"time"

	"github.com/jim-minter/azure-helm/pkg/tunnel/config"
)

type dialer struct {
	*config.Config
}

var _ socket = &dialer{}

func newDialer(config *config.Config) (socket, error) {
	return &dialer{Config: config}, nil
}

func (d *dialer) GetConn() net.Conn {
	for {
		c, err := tls.Dial("tcp4", d.Address, &tls.Config{
			Certificates: []tls.Certificate{
				{
					Certificate: [][]byte{
						d.Cert.Raw,
					},
					PrivateKey: d.Key,
				},
			},
			RootCAs: d.CACertPool,
		})
		if err == nil {
			return c
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func (*dialer) Close() error {
	return nil
}
