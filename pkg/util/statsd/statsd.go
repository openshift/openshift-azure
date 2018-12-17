package statsd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
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
	conn net.Conn
	w    *bufio.Writer
}

// NewClient returns a new client
func NewClient(conn net.Conn) *Client {
	return &Client{conn: conn, w: bufio.NewWriter(conn)}
}

// Flush flushes the internal buffer
func (c *Client) Flush() error {
	return c.w.Flush()
}

// Write writes a gauge to the internal buffer
func (c *Client) Write(g *Gauge) error {
	b, err := g.marshal()
	if err != nil {
		return err
	}

	_, err = c.w.Write(b)
	return err
}

// Close flushes the internal buffer and closes the connection
func (c *Client) Close() error {
	if err := c.Flush(); err != nil {
		return err
	}

	return c.conn.Close()
}
