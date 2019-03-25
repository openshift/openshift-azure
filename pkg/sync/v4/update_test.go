// +build update

// This is the utility to update the data/ directory.  It does not run as a test
// by default.  To run it against a cluster, do `go test -tags update .`

package sync

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

type updater struct{}

// readDB uses the discovery and dynamic clients to read all objects from an API
// server into a map.
func (u *updater) readDB() (map[string]unstructured.Unstructured, error) {
	db := map[string]unstructured.Unstructured{}

	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})

	restconfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	cli, err := discovery.NewDiscoveryClientForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	grs, err := discovery.GetAPIGroupResources(cli)
	if err != nil {
		return nil, err
	}

	rm := discovery.NewRESTMapper(grs, meta.InterfacesForUnstructured)
	dyn := dynamic.NewClientPool(restconfig, rm, dynamic.LegacyAPIPathResolverFunc)

	done := map[schema.GroupKind]struct{}{}
	for _, gr := range grs {
		for version, resources := range gr.VersionedResources {
			for _, resource := range resources {
				if strings.ContainsRune(resource.Name, '/') { // no subresources
					continue
				}

				if !contains(resource.Verbs, "list") {
					continue
				}

				gvk := schema.GroupVersionKind{Group: gr.Group.Name, Version: version, Kind: resource.Kind}
				gk := gvk.GroupKind()
				if isDouble(gk) {
					continue
				}

				if _, found := done[gk]; found {
					continue
				}
				done[gk] = struct{}{}

				dc, err := dyn.ClientForGroupVersionKind(gvk)
				if err != nil {
					return nil, err
				}

				o, err := dc.Resource(&resource, "").List(metav1.ListOptions{})
				if err != nil {
					return nil, err
				}

				l, ok := o.(*unstructured.UnstructuredList)
				if !ok {
					continue
				}

				for _, i := range l.Items {
					db[keyFunc(i.GroupVersionKind().GroupKind(), i.GetNamespace(), i.GetName())] = i
				}
			}
		}
	}

	return db, nil
}

// blank uses translate() to insert a placeholder in all configuration items
// that will be templated upon import, to avoid persisting any secrets.
func (u *updater) blank(o unstructured.Unstructured) (unstructured.Unstructured, error) {
	for _, t := range translations[keyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName())] {
		err := translate(o.Object, t.Path, t.NestedPath, t.nestedFlags, "*** GENERATED ***")
		if err != nil {
			return unstructured.Unstructured{}, err
		}
	}

	return o, nil
}

// writeDB selects, prepares and outputs YAML files for all relevant objects.
func (u *updater) writeDB(db map[string]unstructured.Unstructured) error {
	for _, o := range db {
		if !wants(o) {
			continue
		}

		err := clean(o)
		if err != nil {
			return err
		}

		defaults(o)

		o, err := u.blank(o)
		if err != nil {
			return err
		}

		err = u.writeYAML(o)
		if err != nil {
			return err
		}
	}

	return nil
}

// write outputs a YAML file for a given object.
func (u *updater) writeYAML(o unstructured.Unstructured) error {
	gk := o.GroupVersionKind().GroupKind()
	// we don't support :s in the file name, in order to allow the repo to be
	// checked out on Windows
	name := strings.Replace(o.GetName(), ":", "-", -1)
	p := fmt.Sprintf("data/%s/%s/%s.yaml", gk.String(), o.GetNamespace(), name)

	err := os.MkdirAll(filepath.Dir(p), 0777)
	if err != nil {
		return err
	}

	b, err := yaml.Marshal(o.Object)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(p, b, 0666)
}

func TestUpdate(t *testing.T) {
	var u updater

	db, err := u.readDB()
	if err != nil {
		t.Fatal(err)
	}

	err = os.RemoveAll("data")
	if err != nil {
		t.Fatal(err)
	}

	err = u.writeDB(db)
	if err != nil {
		t.Fatal(err)
	}
}
