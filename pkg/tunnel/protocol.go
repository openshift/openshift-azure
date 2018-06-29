package tunnel

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

const version = 1

type helloMessage struct {
	Version uint16
	Nets    uint16
}

type netMessage struct {
	IP   [4]byte
	Mask [4]byte
}

func handshake(c net.Conn, nets []net.IPNet) ([]net.IPNet, error) {
	err := binary.Write(c, binary.BigEndian, helloMessage{Version: version, Nets: uint16(len(nets))})
	if err != nil {
		return nil, err
	}

	for _, n := range nets {
		var nm netMessage
		copy(nm.IP[:], n.IP.To4())
		copy(nm.Mask[:], net.IP(n.Mask).To4())

		err = binary.Write(c, binary.BigEndian, nm)
		if err != nil {
			return nil, err
		}
	}

	log.Printf("sent nets %v\n", nets)

	var h helloMessage
	err = binary.Read(c, binary.BigEndian, &h)
	if err != nil {
		return nil, err
	}

	if h.Version != version {
		return nil, fmt.Errorf("invalid protocol version %d", h.Version)
	}

	nets = make([]net.IPNet, 0, h.Nets)
	for i := uint16(0); i < h.Nets; i++ {
		var nm netMessage
		err = binary.Read(c, binary.BigEndian, &nm)
		if err != nil {
			return nil, err
		}
		nets = append(nets, net.IPNet{IP: nm.IP[:], Mask: nm.Mask[:]})
	}

	log.Printf("received nets %v\n", nets)

	return nets, nil
}
