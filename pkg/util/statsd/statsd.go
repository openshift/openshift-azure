// Package statsd implements the modified statsd protocol documented at
// https://genevamondocs.azurewebsites.net/collect/references/statsdref.html
package statsd

import (
	"bytes"
	"encoding/json"
	"fmt"
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

// MarshalJSON marshals a Float into JSON format.  You should probably call
// Marshal() instead.
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
