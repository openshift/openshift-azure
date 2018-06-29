package protocol

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/jim-minter/azure-helm/pkg/tunnel/cdb"
	"github.com/jim-minter/azure-helm/pkg/tunnel/config"
)

type protocol struct {
	config *config.Config
	m      sync.Mutex // serialise writers
	c      net.Conn
	d      *gob.Decoder
	e      *gob.Encoder
	done   chan struct{}
}

var _ cdb.PacketReadWriteCloser = &protocol{}

func NewProtocol(config *config.Config, c net.Conn) *protocol {
	return &protocol{
		config: config,
		c:      c,
		d:      gob.NewDecoder(c),
		e:      gob.NewEncoder(c),
		done:   make(chan struct{}),
	}
}

const protocolVersion = 1

type helloMessage struct {
	Version uint8
	Nets    []net.IPNet
}

type heartbeatMessage struct{}

type packetMessage struct {
	Pkt []byte
}

func init() {
	gob.Register(&helloMessage{})
	gob.Register(&heartbeatMessage{})
	gob.Register(&packetMessage{})
}

func (p *protocol) Handshake(nets []net.IPNet) ([]net.IPNet, error) {
	var m interface{}

	m = &helloMessage{Version: protocolVersion, Nets: nets}
	err := p.write(m)
	if err != nil {
		return nil, err
	}

	log.Printf("wrote %#v\n", m)

	m, err = p.read()
	if err != nil {
		return nil, err
	}

	switch m := m.(type) {
	case *helloMessage:
		log.Printf("read %#v\n", m)

		if m.Version != protocolVersion {
			return nil, fmt.Errorf("invalid protocol version %d", m.Version)
		}

		go p.heartbeat()

		return m.Nets, nil

	default:
		return nil, fmt.Errorf("invalid protocol message %T", m)
	}
}

func (p *protocol) heartbeat() (err error) {
	defer func() {
		if err != nil {
			log.Println(err)
		}
	}()

	t := time.NewTicker(p.config.Heartbeat)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			err = p.write(heartbeatMessage{})
			if err != nil {
				return err
			}

		case <-p.done:
			return nil
		}
	}
}

func (p *protocol) read() (interface{}, error) {
	err := p.c.SetReadDeadline(time.Now().Add(p.config.HeartbeatTimeout))
	if err != nil {
		return nil, err
	}

	var i interface{}
	err = p.d.Decode(&i)
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (p *protocol) write(i interface{}) error {
	p.m.Lock()
	defer p.m.Unlock()

	return p.e.Encode(&i)
}

func (p *protocol) ReadPacket() ([]byte, error) {
	for {
		i, err := p.read()
		if err != nil {
			return nil, err
		}

		switch i := i.(type) {
		case *packetMessage:
			return i.Pkt, nil

		case *heartbeatMessage:

		default:
			return nil, fmt.Errorf("invalid protocol message %T", i)
		}
	}
}

func (p *protocol) WritePacket(pkt []byte) error {
	return p.write(packetMessage{Pkt: pkt})
}

func (p *protocol) Close() error {
	close(p.done)
	return nil
}
