package statsd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/sirupsen/logrus"
)

// Gauge represents a statsd gauge
type Gauge struct {
	Namespace string
	Metric    string
	Dims      map[string]string
	Value     float64 `json:"-"`
}

func (g *Gauge) marshal() ([]byte, error) {
	buf := &bytes.Buffer{}

	e := json.NewEncoder(buf)
	err := e.Encode(g)
	if err != nil {
		return nil, err
	}

	// json.Encoder.Encode() appends a "\n" that we don't want - remove it
	if buf.Len() > 1 {
		buf.Truncate(buf.Len() - 1)
	}

	_, err = fmt.Fprintf(buf, ":%f|g\n", g.Value)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Client is a buffering statsd client
type Client struct {
	log        *logrus.Entry
	conn       net.Conn
	buf        bytes.Buffer
	maxPayload int
}

// NewClient returns a new client
func NewClient(log *logrus.Entry, address string) (*Client, error) {
	conn, err := net.Dial("udp", address)
	if err != nil {
		return nil, err
	}

	c := &Client{log: log, conn: conn}

	if err := c.setMaxPayload(); err != nil {
		return nil, err
	}

	return c, nil
}

// Flush flushes the internal buffer
func (c *Client) Flush() error {
	_, err := c.buf.WriteTo(c.conn)
	if err, ok := err.(*net.OpError); ok {
		if err, ok := err.Err.(*os.SyscallError); ok {
			if err.Err == syscall.EMSGSIZE {
				err := c.setMaxPayload()
				if err != nil {
					c.log.Warn(err)
				}
				c.buf.Reset()
			}
		}
	}

	return err
}

// Write writes a gauge to the internal buffer and possibly flushes
func (c *Client) Write(g *Gauge) error {
	b, err := g.marshal()
	if err != nil {
		return err
	}

	if c.buf.Len()+len(b) > c.maxPayload {
		if err := c.Flush(); err != nil {
			return err
		}
	}

	_, err = c.buf.Write(b)
	return err
}

// Close flushes the internal buffer and closes the connection
func (c *Client) Close() error {
	if err := c.Flush(); err != nil {
		return err
	}

	return c.conn.Close()
}

func (c *Client) setMaxPayload() error {
	f, err := c.conn.(*net.UDPConn).File()
	if err != nil {
		return err
	}
	defer f.Close()

	fd := int(f.Fd())

	mtu, err := syscall.GetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_MTU)
	if err != nil {
		return err
	}

	c.maxPayload = mtu - 28 // len(typical IP header) + len(UDP header)

	if c.maxPayload > 2048 {
		c.maxPayload = 2048 // downstream reader may not be able to cope with larger packets *cough*
	}

	c.log.Infof("set maxPayload to %d", c.maxPayload)

	return nil
}
