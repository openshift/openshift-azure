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
	"time"

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
	Heartbeat          time.Duration
	HeartbeatTimeout   time.Duration
	ServicesSubnet     net.IPNet
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
		Heartbeat          int      `json:"heartbeatSeconds"`
		HeartbeatTimeout   int      `json:"heartbeatTimeoutSeconds"`
		ServicesSubnet     string   `json:"servicesSubnet"`
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
		Heartbeat:          time.Duration(ext.Heartbeat) * time.Second,
		HeartbeatTimeout:   time.Duration(ext.HeartbeatTimeout) * time.Second,
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
		n.IP = n.IP.To4()
		cc.AdvertiseCIDRs = append(c.AdvertiseCIDRs, *n)
	}

	// TODO: remove this
	if ext.ServicesSubnet == "" {
		ext.ServicesSubnet = "172.31.0.0/16"
	}
	_, n, err := net.ParseCIDR(ext.ServicesSubnet)
	if err != nil {
		return err
	}
	n.IP = n.IP.To4()
	cc.ServicesSubnet = *n

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

	switch {
	case c.Heartbeat < 0:
		return fmt.Errorf("invalid heartbeatSeconds %q", c.Heartbeat)
	case c.Heartbeat == 0:
		c.Heartbeat = 10 * time.Second
	}

	switch {
	case c.HeartbeatTimeout < 0:
		return fmt.Errorf("invalid heartbeatTimeoutSeconds %q", c.HeartbeatTimeout)
	case c.HeartbeatTimeout == 0:
		c.HeartbeatTimeout = 30 * time.Second
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
