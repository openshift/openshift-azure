package cdb

import (
	"log"
	"net"
	"sync"
)

type PacketReader interface {
	ReadPacket() ([]byte, error)
}

type PacketWriter interface {
	WritePacket([]byte) error
}

type PacketReadWriteCloser interface {
	PacketReader
	PacketWriter
	Close() error
}

type Cdb struct {
	m    sync.Mutex
	nets []cdbnet
}

type cdbnet struct {
	n net.IPNet
	w PacketWriter
}

func (cdb *Cdb) AddNets(nets []net.IPNet, w PacketWriter) {
	cdb.m.Lock()
	defer cdb.m.Unlock()

	for _, n := range nets {
		cdb.nets = append(cdb.nets, cdbnet{n, w})
	}
}

func (cdb *Cdb) DeleteWriter(w PacketWriter) {
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

func (cdb *Cdb) WritePacket(pkt []byte) error {
	cdb.m.Lock()
	defer cdb.m.Unlock()

	src := net.IP(pkt[12:16])
	dst := net.IP(pkt[16:20])

	for _, cdbn := range cdb.nets {
		if cdbn.n.Contains(dst) {
			return cdbn.w.WritePacket(pkt)
		}
	}

	log.Printf("dropped %s->%s %d\n", src.String(), dst.String(), pkt[9])
	return nil
}
