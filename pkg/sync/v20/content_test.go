// +build content

package sync

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	imagev1 "github.com/openshift/api/image/v1"
	templatev1 "github.com/openshift/api/template/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	kjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	err := imagev1.DeprecatedInstallWithoutGroup(scheme.Scheme)
	if err != nil {
		panic(err)
	}
}

type item struct {
	Name        string `json:"name,omitempty"`
	Docs        string `json:"docs,omitempty"`
	SourceURL   string `json:"source_url,omitempty"`
	Description string `json:"description,omitempty"`
	Path        string `json:"path,omitempty"`
}

type index map[string] /*folder*/ map[string] /*item_type*/ []item

func get(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	return ioutil.ReadAll(resp.Body)
}

func writeObject(o runtime.Object, path string) error {
	b, err := json.Marshal(o)
	if err != nil {
		return err
	}

	var u unstructured.Unstructured
	_, _, err = unstructured.UnstructuredJSONScheme.Decode(b, nil, &u)
	if err != nil {
		return err
	}

	err = clean(u)
	if err != nil {
		return err
	}

	b, err = yaml.Marshal(u.Object)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, b, 0666)
}

func handleImageStream(folder string, item *item) (touched []string, err error) {
	var b []byte

	if folder == "jenkins" {
		// upstream issue: Jenkins images are tightly coupled to latest
		// openshift version.  Look for openshift-3.11 images and if we can't
		// find a matching image on that branch, keep on moving.
		item.SourceURL = strings.Replace(item.SourceURL, "master", "openshift-3.11", -1)
		b, err = get(item.SourceURL)
		if err != nil && err.Error() == "unexpected status code 404" {
			return nil, nil
		}

	} else {
		b, err = get(item.SourceURL)
	}
	if err != nil {
		return nil, err
	}

	var iss []imagev1.ImageStream

	s := kjson.NewYAMLSerializer(kjson.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	o, _, err := s.Decode(b, nil, nil)
	if err != nil {
		return nil, err
	}

	switch o := o.(type) {
	case *corev1.List:
		for _, o := range o.Items {
			var is imagev1.ImageStream
			_, _, err := s.Decode(o.Raw, nil, &is)
			if err != nil {
				return nil, err
			}
			iss = append(iss, is)
		}
	case *imagev1.ImageStreamList:
		iss = append(iss, o.Items...)
	case *imagev1.ImageStream:
		iss = append(iss, *o)
	default:
		return nil, fmt.Errorf("unhandled type %T", o)
	}

	for _, is := range iss {
		is.APIVersion = "image.openshift.io/v1"
		is.Namespace = "openshift"

		// upstream files don't sort spec.tags.  Do so to keep the diff simpler.
		sort.Slice(is.Spec.Tags, func(i, j int) bool {
			return is.Spec.Tags[i].Name < is.Spec.Tags[j].Name
		})

		err = writeObject(&is, "data/ImageStream.image.openshift.io/openshift/"+is.Name+".yaml")
		if err != nil {
			return nil, err
		}

		touched = append(touched, is.Name+".yaml")
	}

	return touched, nil
}

func handleImageStreams(index index) error {
	touched := map[string]struct{}{}

	for folder, items := range index {
		for _, item := range items["imagestreams"] {
			files, err := handleImageStream(folder, &item)
			if err != nil {
				return err
			}
			for _, file := range files {
				touched[file] = struct{}{}
			}
		}
	}

	// remove anything we haven't touched.
	f, err := os.Open("data/ImageStream.image.openshift.io/openshift")
	if err != nil {
		return err
	}
	defer f.Close()

	files, err := f.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, file := range files {
		if _, found := touched[file]; !found {
			err = os.Remove("data/ImageStream.image.openshift.io/openshift/" + file)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func handleTemplate(folder string, item *item) (string, error) {
	b, err := get(item.SourceURL)
	if err != nil {
		return "", err
	}

	var t templatev1.Template

	s := kjson.NewYAMLSerializer(kjson.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	_, _, err = s.Decode(b, nil, &t)
	if err != nil {
		return "", err
	}

	t.APIVersion = "template.openshift.io/v1"
	t.Namespace = "openshift"

	err = writeObject(&t, "data/Template.template.openshift.io/openshift/"+t.Name+".yaml")
	if err != nil {
		return "", err
	}

	return t.Name + ".yaml", nil
}

func handleTemplates(index index) error {
	touched := map[string]struct{}{}

	for folder, items := range index {
		for _, item := range items["templates"] {
			file, err := handleTemplate(folder, &item)
			if err != nil {
				return err
			}
			touched[file] = struct{}{}
		}
	}

	// remove anything we haven't touched.
	f, err := os.Open("data/Template.template.openshift.io/openshift")
	if err != nil {
		return err
	}
	defer f.Close()

	files, err := f.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, file := range files {
		if _, found := touched[file]; !found {
			// project-request is not part of content-update flow
			if file == "project-request.yaml" {
				continue
			}
			// strimzi templates are retrieved from a different source (access.redhat.com)
			if strings.Contains(file, "strimzi-") {
				continue
			}
			err = os.Remove("data/Template.template.openshift.io/openshift/" + file)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func TestContent(t *testing.T) {
	b, err := get("https://github.com/openshift/library/raw/master/official/index.json")
	if err != nil {
		t.Fatal(err)
	}

	var index index
	err = json.Unmarshal(b, &index)
	if err != nil {
		t.Fatal(err)
	}

	err = handleImageStreams(index)
	if err != nil {
		t.Fatal(err)
	}

	err = handleTemplates(index)
	if err != nil {
		t.Fatal(err)
	}
}
