package tunnel

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
)

type packetConn struct {
	net.Conn
}

var _ io.ReadWriteCloser = &packetConn{}

func (c *packetConn) Read(b []byte) (int, error) {
	if len(b) < 65536 {
		return 0, errors.New("read buffer too short")
	}

	n, err := io.ReadFull(c.Conn, b[:20])
	if err != nil {
		return n, err
	}

	l := binary.BigEndian.Uint16(b[2:4])

	n2, err := io.ReadFull(c.Conn, b[20:l])
	return n + n2, err
}
