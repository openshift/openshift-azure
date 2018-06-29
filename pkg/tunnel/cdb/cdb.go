package cdb

import (
	"io"
	"log"
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
	cdb.m.Lock()
	defer cdb.m.Unlock()

	src := net.IP(pkt[12:16])
	dst := net.IP(pkt[16:20])

	for _, cdbn := range cdb.nets {
		if cdbn.n.Contains(dst) {
			return cdbn.w.Write(pkt)
		}
	}

	log.Printf("dropped %s->%s %d\n", src.String(), dst.String(), pkt[9])
	return 0, nil
}
