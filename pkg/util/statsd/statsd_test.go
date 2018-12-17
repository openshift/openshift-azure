package statsd

import (
	"testing"
	"time"
)

func TestMarshal(t *testing.T) {
	f := Float{
		Metric:    "metric",
		Namespace: "namespace",
		Dims:      map[string]string{"key": "value"},
		Value:     1.0,
		TS:        time.Unix(0, 0),
	}
	b, err := f.marshal()
	if err != nil {
		t.Fatal(err)
	}

	if string(b) != `{"Metric":"metric","Namespace":"namespace","Dims":{"key":"value"},"TS":"1970-01-01T00:00:00.000"}:1.000000|f`+"\n" {
		t.Errorf("unexpected marshal output %s", string(b))
	}
}
