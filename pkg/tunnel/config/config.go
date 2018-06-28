package config

import (
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"syscall"

	"github.com/ghodss/yaml"
)

type Config struct {
	Mode               string
	Address            string
	Interface          string
	CACertPool         *x509.CertPool
	Cert               *x509.Certificate
	Key                crypto.PrivateKey
	ClientOrganization string
	AdvertiseCIDRs     []net.IPNet
}

func (c *Config) UnmarshalJSON(b []byte) error {
	ext := struct {
		Mode               string   `json:"mode"`
		Address            string   `json:"address"`
		Interface          string   `json:"interface"`
		KeyPath            string   `json:"keyPath"`
		CertPath           string   `json:"certPath"`
		CACertPath         string   `json:"caCertPath"`
		ClientOrganization string   `json:"clientOrganization"`
		AdvertiseCIDRs     []string `json:"advertiseCIDRs"`
	}{}

	err := json.Unmarshal(b, &ext)
	if err != nil {
		return err
	}

	cc := Config{
		Mode:               ext.Mode,
		Address:            ext.Address,
		Interface:          ext.Interface,
		CACertPool:         x509.NewCertPool(),
		ClientOrganization: ext.ClientOrganization,
	}

	cc.AdvertiseCIDRs = make([]net.IPNet, 0, len(ext.AdvertiseCIDRs))
	for _, cidr := range ext.AdvertiseCIDRs {
		if len(cidr) == 0 {
			continue
		}
		if cidr[0] < '0' || cidr[0] > '9' {
			slash := strings.IndexByte(cidr, '/')
			if slash != -1 {
				ip, err := getInterfaceIP(cidr[:slash])
				if err != nil {
					return err
				}
				cidr = ip.String() + cidr[slash:]
			}
		}
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}
		cc.AdvertiseCIDRs = append(c.AdvertiseCIDRs, *n)
	}

	cc.Key, err = readKey(ext.KeyPath)
	if err != nil {
		return err
	}

	cc.Cert, err = readCert(ext.CertPath)
	if err != nil {
		return err
	}

	cacert, err := readCert(ext.CACertPath)
	if err != nil {
		return err
	}
	cc.CACertPool.AddCert(cacert)

	*c = cc

	return nil
}

func (c *Config) Validate() error {
	switch c.Mode {
	case "client", "server":
	default:
		return fmt.Errorf("invalid mode %q", c.Mode)
	}

	if c.Mode == "client" && c.ClientOrganization != "" {
		return fmt.Errorf("clientOrganization can't be set in client mode")
	}

	_, err := getInterfaceIP(c.Interface)
	if e, ok := err.(syscall.Errno); !ok || e != syscall.ENODEV {
		return fmt.Errorf("interface %q already exists", c.Interface)
	}

	return nil
}

func Read(filename string) (*Config, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var c Config
	err = yaml.Unmarshal(b, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func readKey(filename string) (crypto.PrivateKey, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	for {
		var block *pem.Block
		block, b = pem.Decode(b)

		if block == nil {
			return nil, errors.New("couldn't find PEM key block")
		}

		switch block.Type {
		case "RSA PRIVATE KEY":
			return x509.ParsePKCS1PrivateKey(block.Bytes)
		case "EC PRIVATE KEY":
			return x509.ParseECPrivateKey(block.Bytes)
		}
	}
}

func readCert(filename string) (*x509.Certificate, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	for {
		var block *pem.Block
		block, b = pem.Decode(b)

		if block == nil {
			return nil, errors.New("couldn't find PEM certificate block")
		}

		switch block.Type {
		case "CERTIFICATE":
			return x509.ParseCertificate(block.Bytes)
		}
	}
}
