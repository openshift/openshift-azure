package tunnel

import (
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/jim-minter/azure-helm/pkg/tunnel/cdb"
	"github.com/jim-minter/azure-helm/pkg/tunnel/config"
	"github.com/jim-minter/azure-helm/pkg/tunnel/protocol"
	"github.com/jim-minter/azure-helm/pkg/tunnel/routes"
	"github.com/jim-minter/azure-helm/pkg/tunnel/tun"
)

func forwarder(cdb *cdb.Cdb, r cdb.PacketReader) error {
	for {
		pkt, err := r.ReadPacket()
		if err != nil {
			return err
		}

		if len(pkt) < 1 || pkt[0]&0xF0 != 0x40 {
			continue // not IPv4
		}

		err = cdb.WritePacket(pkt)
		if err != nil {
			return err
		}
	}
}

func handleConn(config *config.Config, cdb *cdb.Cdb, c net.Conn) (err error) {
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	p := protocol.NewProtocol(config, c)
	defer p.Close()

	remotenets, err := p.Handshake(config.AdvertiseCIDRs)
	if err != nil {
		return err
	}

	cdb.AddNets(remotenets, p)
	defer cdb.DeleteWriter(p)

	err = routes.Add(remotenets, config.Interface)
	if err != nil {
		return err
	}
	defer routes.Delete(remotenets, config.Interface)

	return forwarder(cdb, p)
}

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

	tun, err := tun.NewTun(config.Interface)
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
		if hostname == "vm-infra-0" || hostname == "infra-000000" {
			config.AdvertiseCIDRs = append(config.AdvertiseCIDRs, config.ServicesSubnet)
		}
	}

	cdb := &cdb.Cdb{}
	cdb.AddNets(config.AdvertiseCIDRs, tun)

	go forwarder(cdb, tun)

	switch config.Mode {
	case "server":
		l, err := listen(config)
		if err != nil {
			return err
		}

		for {
			c, err := accept(config, l)
			if err != nil {
				if err != io.EOF { // don't log LB probes which don't negotiate TLS
					log.Println(err)
				}
			} else {
				go handleConn(config, cdb, c)
			}
		}

	case "client":
		for {
			c, err := dial(config)
			if err != nil {
				log.Println(err)
			} else {
				handleConn(config, cdb, c)
			}

			time.Sleep(time.Second)
		}
	}

	return nil
}
