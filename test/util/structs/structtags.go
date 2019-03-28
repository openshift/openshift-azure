package structs

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

// CheckJSONTags checks that the json tags on a struct are present and in lower camel case format and
// reports violations as errors. The function will only produce meaningful results when a struct
// is passed in. It returns a nil empty slice when passed other types of values.
func CheckJsonTags(o interface{}) (errs []error) {
	if o != nil {
		t := reflect.ValueOf(o)
		return checkJSONTags(t)
	}
	return
}

func checkJSONTags(t reflect.Value) (errs []error) {
	switch t.Kind() {
	case reflect.Ptr:
		return append(errs, checkJSONTags(t.Elem())...)
	case reflect.Struct:
		// do not go on with the recursion if it isn't one of the core openshift-azure types
		if !strings.HasPrefix(t.Type().PkgPath(), "github.com/openshift/openshift-azure/") ||
			strings.HasPrefix(t.Type().PkgPath(), "github.com/openshift/openshift-azure/vendor/") {
			return
		}
		for i := 0; i < t.NumField(); i++ {
			f := t.Type().Field(i)
			tag := f.Tag.Get("json")
			if tag == "" {
				msg := fmt.Sprintf(`field "%v" does not have a json tag`, f.Name)
				errs = append(errs, errors.New(msg))
				continue
			}
			parts := strings.Split(tag, ",")
			if len(parts) > 1 && parts[1] != "omitempty" {
				msg := fmt.Sprintf("invalid tag %s", tag)
				errs = append(errs, errors.New(msg))
				continue
			}
			availableTag := parts[0]
			computedTag := toTag(f.Name)
			if availableTag == "-" {
				continue
			}
			if computedTag != availableTag {
				errs = append(errs, fmt.Errorf("%s: tag name mismatch: wanted %s, got %s", f.Name, computedTag, availableTag))
			}
			errs = append(errs, checkJSONTags(t.Field(i))...)
		}
	}
	return
}

// toTag converts a struct field name like ImageSKU to imageSku
func toTag(s string) string {
	switch strings.TrimSpace(s) {
	case "":
		return s
	case "VnetSubnetID":
		// should theoretically be vnetSubnetId, but isn't.  Our behaviour here
		// matches the AKS API.
		return "vnetSubnetID"
	}

	s = strings.TrimPrefix(s, "Deprecated")

	for _, acronym := range []string{"API", "CIDR", "FQDN", "HTTP", "ID", "SDN", "SKU", "SSH", "TLS", "VM"} {
		lower := string(acronym[0]) + strings.Map(unicode.ToLower, acronym[1:])
		s = strings.Replace(s, acronym, lower, -1)
	}
	return string(unicode.ToLower(rune(s[0]))) + s[1:]
}
