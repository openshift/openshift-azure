package tunnel

import (
	"io"
	"log"
	"net"
	"os"

	"github.com/jim-minter/azure-helm/pkg/tunnel/cdb"
	"github.com/jim-minter/azure-helm/pkg/tunnel/config"
	"github.com/jim-minter/azure-helm/pkg/tunnel/routes"
	"github.com/jim-minter/azure-helm/pkg/tunnel/tun"
)

type socket interface {
	GetConn() net.Conn
	Close() error
}

func forwarder(r io.Reader, cdb *cdb.Cdb) error {
	for {
		pkt := make([]byte, 65536)
		n, err := r.Read(pkt)
		if err != nil {
			log.Println(err)
			return err
		}
		pkt = pkt[:n]

		if len(pkt) < 20 {
			continue // too short
		}

		if pkt[0]&0xF0 != 0x40 {
			continue // not IPv4
		}

		_, err = cdb.Write(pkt)
		if err != nil {
			log.Println(err)
			return err
		}
	}
}

func handleConn(config *config.Config, cdb *cdb.Cdb, c net.Conn) error {
	defer c.Close()

	remotenets, err := handshake(c, config.AdvertiseCIDRs)
	if err != nil {
		return err
	}

	cdb.AddNets(remotenets, c)
	defer cdb.DeleteConn(c)

	err = routes.Add(remotenets, config.Interface)
	if err != nil {
		return err
	}
	defer routes.Delete(remotenets, config.Interface)

	return forwarder(&packetConn{Conn: c}, cdb)
}

var servicesSubnet = net.IPNet{IP: net.ParseIP("172.31.0.0"), Mask: net.CIDRMask(16, 32)}

func Run() error {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	config, err := config.Read(os.Args[1])
	if err != nil {
		return err
	}

	err = config.Validate()
	if err != nil {
		return err
	}

	tun, err := tun.Tun(config.Interface)
	if err != nil {
		return err
	}
	defer tun.Close()

	// TODO: fix this gross hack
	if config.Mode == "client" {
		hostname, err := os.Hostname()
		if err != nil {
			return err
		}
		if hostname == "vm-infra-0" {
			config.AdvertiseCIDRs = append(config.AdvertiseCIDRs, servicesSubnet)
		}
	}

	cdb := &cdb.Cdb{Config: config}
	cdb.AddNets(config.AdvertiseCIDRs, tun)

	go forwarder(tun, cdb)

	var s socket
	switch config.Mode {
	case "server":
		s, err = newListener(config)
		if err != nil {
			return err
		}

		for {
			go handleConn(config, cdb, s.GetConn())
		}

	case "client":
		s, err = newDialer(config)
		if err != nil {
			return err
		}

		for {
			handleConn(config, cdb, s.GetConn())
		}
	}

	return nil
}
