package sync

import (
	"io/ioutil"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/test/util/populate"
)

func getObjectFromFile(path string) *unstructured.Unstructured {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	o, err := unmarshal(b)
	if err != nil {
		panic(err)
	}
	return &o
}

// TODO: We need fuzz testing
func TestGetHash(t *testing.T) {
	tests := []struct {
		name  string
		i1    *unstructured.Unstructured
		i2    *unstructured.Unstructured
		match bool
	}{
		{
			name:  "same object matches",
			i1:    getObjectFromFile("testdata/secret1.yaml"),
			i2:    getObjectFromFile("testdata/secret1.yaml"),
			match: true,
		},
		{
			name:  "different objects do not match",
			i1:    getObjectFromFile("testdata/secret1.yaml"),
			i2:    getObjectFromFile("testdata/secret2.yaml"),
			match: false,
		},
		{
			name:  "semantically same objects match",
			i1:    getObjectFromFile("testdata/secret1.yaml"),
			i2:    getObjectFromFile("testdata/secret3.yaml"),
			match: true,
		},
	}

	for _, test := range tests {
		first := getHash(test.i1)
		sec := getHash(test.i2)
		if test.match && first != sec {
			t.Errorf("%s: expected hashes to match", test.name)
		}
		if !test.match && first == sec {
			t.Errorf("%s: unexpected hashes match", test.name)
		}
	}
}

func TestReadDB(t *testing.T) {
	prepare := func(v reflect.Value) {
		switch v.Interface().(type) {
		case []api.IdentityProvider:
			// set the Provider to AADIdentityProvider
			v.Set(reflect.ValueOf([]api.IdentityProvider{{Provider: &api.AADIdentityProvider{Kind: "AADIdentityProvider"}}}))
		}
	}

	var cs api.OpenShiftManagedCluster
	populate.Walk(&cs, prepare)
	cs.ID = "subscriptions/foo/resourceGroups/bar/providers/baz/qux/quz"
	cs.Config.ImageVersion = "311.123.456"
	cs.Config.Images.Console = "foo:v3.11.22"
	cs.Config.Images.PrometheusOperator = ":"
	cs.Config.Images.PrometheusConfigReloader = ":"
	cs.Config.Images.ConfigReloader = ":"
	cs.Config.Images.Prometheus = ":"
	cs.Config.Images.AlertManager = ":"
	cs.Config.Images.Grafana = ":"
	cs.Config.Images.OAuthProxy = ":"
	cs.Config.Images.NodeExporter = ":"
	cs.Config.Images.KubeStateMetrics = ":"
	cs.Config.Images.KubeRbacProxy = ":"
	cs.Config.Images.LogAnalyticsAgent = ":"

	s := &sync{cs: &cs}
	err := s.readDB()
	if err != nil {
		t.Error(err)
	}
}
