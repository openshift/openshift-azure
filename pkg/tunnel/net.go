package tunnel

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"net"

	"github.com/jim-minter/azure-helm/pkg/tunnel/config"
)

func accept(config *config.Config, l net.Listener) (net.Conn, error) {
	c, err := l.Accept()
	if err != nil {
		return nil, err
	}

	tc := tls.Server(c, &tls.Config{
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
			return errors.New("certificate did not match clientOrganization")
		},
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  config.CACertPool,
	})

	err = tc.Handshake()
	if err != nil {
		return nil, err
	}

	log.Printf("accepted connection from %s", c.RemoteAddr())

	return tc, nil
}

func dial(config *config.Config) (net.Conn, error) {
	c, err := net.Dial("tcp4", config.Address)
	if err != nil {
		return nil, err
	}

	serverName, _, err := net.SplitHostPort(config.Address)
	if err != nil {
		return nil, err
	}

	tc := tls.Client(c, &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{
					config.Cert.Raw,
				},
				PrivateKey: config.Key,
			},
		},
		RootCAs:    config.CACertPool,
		ServerName: serverName,
	})

	err = tc.Handshake()
	if err != nil {
		return nil, err
	}

	log.Printf("connected to %s", c.RemoteAddr())
	return tc, nil
}

func listen(config *config.Config) (net.Listener, error) {
	log.Printf("listening on %s", config.Address)

	return net.Listen("tcp4", config.Address)
}
