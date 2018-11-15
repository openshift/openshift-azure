package api

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/go-test/deep"

	"github.com/openshift/openshift-azure/test/util/populate"
)

func TestDeepCopy(t *testing.T) {
	prepare := func(v reflect.Value) {
		switch v.Interface().(type) {
		case []IdentityProvider:
			// set the Provider to AADIdentityProvider
			v.Set(reflect.ValueOf([]IdentityProvider{{Provider: &AADIdentityProvider{Kind: "AADIdentityProvider"}}}))
		}
	}

	var cs *OpenShiftManagedCluster
	populate.Walk(&cs, prepare)

	copy := cs.DeepCopy()
	if !reflect.DeepEqual(cs, copy) {
		t.Errorf("OpenShiftManagedCluster differed after DeepCopy: %s", deep.Equal(cs, copy))
	}
	copy.Tags["test"] = "updated"
	copy.Config.HtPasswd[0] = 0
	if _, found := cs.Tags["test"]; found {
		t.Error("unexpectedly found test tag")
	}
	if !bytes.Equal(cs.Config.HtPasswd, []byte("Config.HtPasswd")) {
		t.Error("cs.Config.HtPasswd unexpectedly changed")
	}
}
