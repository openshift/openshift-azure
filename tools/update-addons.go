package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/jim-minter/azure-helm/pkg/addons"
)

var (
	restconfig *rest.Config
)

// getClients populates the Kubernetes client object(s).
func getClients() (err error) {
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{})

	restconfig, err = kubeconfig.ClientConfig()
	return
}

// readDB uses the discovery and dynamic clients to read all objects from an API
// server into a map.
func readDB() (map[string]unstructured.Unstructured, error) {
	db := map[string]unstructured.Unstructured{}

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

	for _, gr := range grs {
		gv, err := schema.ParseGroupVersion(gr.Group.PreferredVersion.GroupVersion)
		if err != nil {
			return nil, err
		}

		for _, resource := range gr.VersionedResources[gr.Group.PreferredVersion.Version] {
			if strings.ContainsRune(resource.Name, '/') { // no subresources
				continue
			}

			if !contains(resource.Verbs, "list") {
				continue
			}

			dc, err := dyn.ClientForGroupVersionKind(gv.WithKind(resource.Kind))
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
				db[addons.KeyFunc(i.GroupVersionKind().GroupKind(), i.GetNamespace(), i.GetName())] = i
			}
		}
	}

	return db, nil
}

// contains returns true if haystack contains needle.
func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// blank uses addons.Translate() to insert a placeholder in all configuration
// items that will be templated upon import, to avoid persisting any secrets.
func blank(o unstructured.Unstructured) (unstructured.Unstructured, error) {
	for _, t := range addons.Translations[addons.KeyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName())] {
		err := addons.Translate(o.Object, t.Path, t.NestedPath, t.NestedFlags, "*** GENERATED ***")
		if err != nil {
			return unstructured.Unstructured{}, err
		}
	}

	return o, nil
}

// writeDB selects, prepares and outputs YAML files for all relevant objects.
func writeDB(db map[string]unstructured.Unstructured) error {
	for _, o := range db {
		if !addons.Wants(o) {
			continue
		}

		err := addons.Clean(o)
		if err != nil {
			return err
		}

		addons.Default(o)

		o, err := blank(o)
		if err != nil {
			return err
		}

		err = write(o)
		if err != nil {
			return err
		}
	}

	return nil
}

// write outputs a YAML file for a given object.
func write(o unstructured.Unstructured) error {
	gk := o.GroupVersionKind().GroupKind()
	p := fmt.Sprintf("pkg/addons/data/%s/%s/%s.yaml", gk.String(), o.GetNamespace(), o.GetName())

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

func main() {
	err := os.RemoveAll("pkg/addons/data")
	if err != nil {
		panic(err)
	}

	err = getClients()
	if err != nil {
		panic(err)
	}

	db, err := readDB()
	if err != nil {
		panic(err)
	}

	err = writeDB(db)
	if err != nil {
		panic(err)
	}
}
