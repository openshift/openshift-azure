package sync

/*
 * This test checks for patterns indicating that environment variables may be
 * used to pass down secrets. The code walks all yaml config files from
 * pkg/sync/<latest>/data and looks for known contructs that currently store
 * that info. For false positives, a whitelist is implemented.
 *
 * The intention is to split the search patterns into separate routines so that
 * it is simple to add or remove work on the data directory looking for those
 * patterns we ultimately want to replace. We probably want to use KeyVault instead.
 */
import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/openshift/openshift-azure/pkg/util/jsonpath"
)

// Blessed files (full path match)
var whitelistWhole = []string{
	"data/DaemonSet.apps/default/docker-registry.yaml",
	"data/DaemonSet.apps/default/router.yaml",
	"data/DaemonSet.apps/openshift-azure-logging/omsagent.yaml",
	"data/DaemonSet.apps/openshift-azure-monitoring/etcd-metrics.yaml", // FIXME we control this
	"data/Deployment.apps/openshift-azure-logging/omsagent-rs.yaml",
	"data/Deployment.apps/default/registry-console.yaml",
}

// Blessed files (path contains pattern)
var whiteListContains = []string{
	"Template.template.openshift.io",
}

func isWhiteListed(path string) bool {
	for _, v := range whiteListContains {
		if strings.Contains(path, v) {
			return true
		}
	}
	for _, v := range whitelistWhole {
		if path == v {
			return true
		}
	}
	return false
}

// We are looking for a structure like this:
// - name: REGISTRY_HTTP_SECRET
//          valueFrom:
//            secretKeyRef:
func checkSecretKeyRef(path string, o unstructured.Unstructured) bool {
	target := jsonpath.MustCompile("$.spec.template.spec.containers[*].env[*].valueFrom.secretKeyRef.key").Get(o.Object)
	for _, v := range target {
		s, ok := v.(string)
		if ok {
			if s == "password" {
				return true
			}
		}
	}
	return false
}

// We are looking for a structure like this:
// env:
// - name: SomeVarName
//   value: '*** GENERATED ***'
func checkValueGenerated(path string, o unstructured.Unstructured, t *testing.T) bool {
	target := jsonpath.MustCompile("$.spec.template.spec.containers[*].env[*].value").Get(o.Object)
	for _, v := range target {
		s, ok := v.(string)
		if ok {
			if s == "*** GENERATED ***" {
				return true
			}
		}
	}
	return false
}

func TestEnvPass(t *testing.T) {
	var secErrs []error

	err := filepath.Walk("data", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if isWhiteListed(path) {
			t.Logf("%s should have passwords passed in via secrets", path)
			return nil
		}

		b1, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		u, err := unmarshal(b1)
		if err != nil {
			return err
		}

		if r := checkSecretKeyRef(path, u); r == true {
			secErrs = append(secErrs, fmt.Errorf("%s: contains 'password' in env[*].valueFrom.secretKeyRef.key which probably means that secrets are passed in environment variables", path))
		}
		if r := checkValueGenerated(path, u, t); r == true {
			secErrs = append(secErrs, fmt.Errorf("%s: contains '*** GENERATED ***' in env[*].value which probably means that secrets are passed in environment variables", path))
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(secErrs) > 0 {
		msg := "The following errors are not allowed for security reasons:"
		for _, sErr := range secErrs {
			msg += "\n" + sErr.Error()
		}
		t.Errorf(msg)
	}
}
