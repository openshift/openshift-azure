package populate

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/openshift/openshift-azure/test/util/tls"
)

type walker struct {
	prepare func(v reflect.Value)
}

// Walk is a recursive struct value population function. Given a pointer to an arbitrarily complex value v, it fills
// in the complete structure of that value, setting each string with the path taken to reach it. An optional prepare
// function may be supplied by the caller of Walk. If supplied, prepare will be called prior to walking v. The prepare
// function is useful for setting custom values to certain fields before walking v.
//
// This function has the following caveats:
//  - Signed integers are set to int(1)
//  - Unsigned integers are set to uint(1)
//  - Floating point numbers are set to float(1.0)
//  - Booleans are set to True
//  - Arrays and slices are allocated 1 element
//  - Maps are allocated 1 element
//  - Only map[string][string] types are supported
//  - strings are set to the value of the path taken to reach the string
func Walk(v interface{}, prepare func(v reflect.Value)) {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		panic("argument is not a pointer to a value")
	}
	walker{prepare: prepare}.walk(val, "")
}

// walk fills in the complete structure of a complex value v using path as the root of the labelling.
func (w walker) walk(v reflect.Value, path string) {
	if !v.IsValid() {
		return
	}

	// special cases
	switch v.Interface().(type) {
	case []byte:
		v.SetBytes([]byte(path))
		return
	case *rsa.PrivateKey:
		// use a dummy value because the zero value cannot be marshalled
		v.Set(reflect.ValueOf(tls.GetDummyPrivateKey()))
		return
	case *x509.Certificate:
		// use a dummy value because the zero value cannot be unmarshalled
		v.Set(reflect.ValueOf(tls.GetDummyCertificate()))
		return
	}

	// call the prepare function, if any, supplied by the client of this function
	if w.prepare != nil {
		w.prepare(v)
	}

	switch v.Kind() {
	case reflect.Interface:
		w.walk(v.Elem(), path)
	case reflect.Ptr:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		w.walk(v.Elem(), path)
	case reflect.Struct:
		// do not go on with the recursion if it isn't one of the core openshift-azure types
		if !strings.HasPrefix(v.Type().PkgPath(), "github.com/openshift/openshift-azure/") ||
			strings.HasPrefix(v.Type().PkgPath(), "github.com/openshift/openshift-azure/vendor/") {
			return
		}
		for i := 0; i < v.NumField(); i++ {
			// do not walk AADIdentityProvider.Kind to prevent breaking AADIdentityProvider unmarshall
			if v.Type().Field(i).Name == "Kind" {
				continue
			}
			field := v.Field(i)
			newpath := extendPath(path, v.Type().Field(i).Name, v.Kind())
			w.walk(field, newpath)
		}
	case reflect.Slice:
		// if the slice has length 0 allocate a new slice of length 1
		if v.Len() == 0 {
			v.Set(reflect.MakeSlice(v.Type(), 1, 1))
		}
		fallthrough
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			field := v.Index(i)
			newpath := extendPath(path, strconv.Itoa(i), v.Kind())
			w.walk(field, newpath)
		}
	case reflect.Map:
		// only map[string]string types are supported
		if v.Type().Key().Kind() != reflect.String || v.Type().Elem().Kind() != reflect.String {
			return
		}
		v.Set(reflect.MakeMap(v.Type()))
		v.SetMapIndex(reflect.ValueOf(path+".key"), reflect.ValueOf(path+".val"))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		v.SetUint(1)
	case reflect.Float32, reflect.Float64:
		v.SetFloat(1.0)
	case reflect.Bool:
		v.SetBool(true)
	case reflect.String:
		v.SetString(path)
	default:
		panic("unimplemented: " + v.Kind().String())
	}
}

// extendPath takes a path and a proposed extension to that path and returns a new path based on the kind of value for which
// the new path is being constructed
func extendPath(path, extension string, kind reflect.Kind) string {
	if path == "" {
		return extension
	}
	switch kind {
	case reflect.Struct:
		return fmt.Sprintf("%s.%s", path, extension)
	case reflect.Slice, reflect.Array:
		return fmt.Sprintf("%s[%s]", path, extension)
	}
	return ""
}
