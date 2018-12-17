// Package statsd implements the modified statsd protocol documented at
// https://genevamondocs.azurewebsites.net/collect/references/statsdref.html
package statsd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Float represents a statsd floating point number
type Float struct {
	Metric    string
	Account   string
	Namespace string
	Dims      map[string]string
	TS        time.Time
	Value     float64
}

func (f *Float) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Metric    string
		Account   string `json:"Account,omitempty"`
		Namespace string `json:"Namespace,omitempty"`
		Dims      map[string]string
		TS        string
	}{
		Metric:    f.Metric,
		Account:   f.Account,
		Namespace: f.Namespace,
		Dims:      f.Dims,
		TS:        f.TS.UTC().Format("2006-01-02T15:04:05.000"),
	})
}

// Marshal a Float into its statsd format.  Call this instead of MarshalJSON().
func (f *Float) Marshal() ([]byte, error) {
	buf := &bytes.Buffer{}

	e := json.NewEncoder(buf)
	err := e.Encode(f)
	if err != nil {
		return nil, err
	}

	// json.Encoder.Encode() appends a "\n" that we don't want - remove it
	if buf.Len() > 1 {
		buf.Truncate(buf.Len() - 1)
	}

	_, err = fmt.Fprintf(buf, ":%f|f\n", f.Value)
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

// Write writes to the internal buffer
func (c *Client) Write(b []byte) (int, error) {
	return c.w.Write(b)
}

// Close flushes the internal buffer and closes the connection
func (c *Client) Close() error {
	if err := c.Flush(); err != nil {
		return err
	}

	return c.conn.Close()
}
