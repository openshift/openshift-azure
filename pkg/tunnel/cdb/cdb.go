package cdb

import (
	"io"
	"net"
	"sync"

	"github.com/jim-minter/azure-helm/pkg/tunnel/config"
)

type Cdb struct {
	Config *config.Config
	m      sync.Mutex
	nets   []cdbnet
}

type cdbnet struct {
	n net.IPNet
	w io.Writer
}

func (cdb *Cdb) AddNets(nets []net.IPNet, w io.Writer) {
	cdb.m.Lock()
	defer cdb.m.Unlock()

	for _, n := range nets {
		cdb.nets = append(cdb.nets, cdbnet{n, w})
	}
}

func (cdb *Cdb) DeleteConn(w io.Writer) {
	cdb.m.Lock()
	defer cdb.m.Unlock()

	newNets := make([]cdbnet, 0, len(cdb.nets))
	for _, cdbn := range cdb.nets {
		if cdbn.w != w {
			newNets = append(newNets, cdbn)
		}
	}
	cdb.nets = newNets
}

func (cdb *Cdb) Write(pkt []byte) (int, error) {
	ip := net.IP(pkt[16:20])

	cdb.m.Lock()
	defer cdb.m.Unlock()

	for _, cdbn := range cdb.nets {
		if cdbn.n.Contains(ip) {
			return cdbn.w.Write(pkt)
		}
	}

	return 0, nil
}
