package structs

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Azure/go-autorest/autorest/to"
)

// walk recursively returns a list of field names contained in a given type.
// For fields of interface type, it iterates a list of candidate types which
// fulfil the interface, passed in in imap.
// TODO: pkg/test/util/populate should be refactored to use an imap argument
// too.
func Walk(t reflect.Type, imap map[string][]reflect.Type) []string {
	var walk func(reflect.Type, string) []string

	walk = func(t reflect.Type, path string) (fields []string) {
		if t.PkgPath() != "" &&
			(!strings.HasPrefix(t.PkgPath(), "github.com/openshift/openshift-azure/") ||
				strings.HasPrefix(t.PkgPath(), "github.com/openshift/openshift-azure/vendor/")) {
			fields = append(fields, path)
			return
		}

		switch t.Kind() {
		case reflect.Struct:
			for i := 0; i < t.NumField(); i++ {
				fields = append(fields, walk(t.Field(i).Type, path+"."+t.Field(i).Name)...)
			}
		case reflect.Ptr, reflect.Slice:
			fields = append(fields, walk(t.Elem(), path)...)
		case reflect.Interface:
			if _, found := imap[path]; !found {
				panic(fmt.Sprintf("imap[%s] not found", path))
			}
			for _, t := range imap[path] {
				fields = append(fields, walk(t, path)...)
			}
		case reflect.Map:
			if (t.Key() != reflect.TypeOf("") && t.Key() != reflect.TypeOf(to.StringPtr(""))) ||
				(t.Elem() != reflect.TypeOf("") && t.Elem() != reflect.TypeOf(to.StringPtr(""))) {
				panic(fmt.Sprintf("unimplemented map type %s", t))
			}
		case reflect.Bool, reflect.Int, reflect.Int64, reflect.String, reflect.Uint8:
			fields = append(fields, path)
		default:
			panic(fmt.Sprintf("unimplemented kind %s", t.Kind()))
		}
		return
	}
	return walk(t, "")
}
