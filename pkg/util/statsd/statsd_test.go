package statsd

import (
	"testing"
)

func TestMarshal(t *testing.T) {
	g := Gauge{
		Namespace: "namespace",
		Metric:    "metric",
		Dims:      map[string]string{"key": "value"},
		Value:     1.0,
	}
	b, err := g.marshal()
	if err != nil {
		t.Fatal(err)
	}

	if string(b) != `{"Namespace":"namespace","Metric":"metric","Dims":{"key":"value"}}:1.000000|g`+"\n" {
		t.Errorf("unexpected marshal output %s", string(b))
	}
}
