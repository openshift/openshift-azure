package statsd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"

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
	log  *logrus.Entry
	conn net.Conn
	buf  bytes.Buffer
}

// NewClient returns a new client
func NewClient(log *logrus.Entry, conn net.Conn) *Client {
	return &Client{log: log, conn: conn}
}

// Flush flushes the internal buffer
func (c *Client) Flush() error {
	_, err := c.buf.WriteTo(c.conn)
	return err
}

// Write writes a gauge to the internal buffer
func (c *Client) Write(g *Gauge) error {
	b, err := g.marshal()
	if err != nil {
		return err
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
