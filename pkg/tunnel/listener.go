package tunnel

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"net"
	"time"

	"github.com/jim-minter/azure-helm/pkg/tunnel/config"
)

type listener struct {
	net.Listener
}

var _ socket = &listener{}

func newListener(config *config.Config) (socket, error) {
	l := &listener{}

	var err error
	l.Listener, err = tls.Listen("tcp4", config.Address, &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{
					config.Cert.Raw,
				},
				PrivateKey: config.Key,
			},
		},
		VerifyPeerCertificate: func(_ [][]byte, verifiedChains [][]*x509.Certificate) error {
			if config.ClientOrganization == "" {
				return nil
			}
			if len(verifiedChains[0][0].Subject.Organization) == 1 &&
				verifiedChains[0][0].Subject.Organization[0] == config.ClientOrganization {
				return nil
			}
			err := errors.New("certificate did not match clientOrganization")
			log.Println(err)
			return err
		},
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  config.CACertPool,
	})
	if err != nil {
		return nil, err
	}

	return l, nil
}

func (l *listener) GetConn() net.Conn {
	for {
		c, err := l.Listener.Accept()
		if err == nil {
			return c
		}

		time.Sleep(100 * time.Millisecond)
	}
}
