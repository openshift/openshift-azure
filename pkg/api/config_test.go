package api_test

import (
	"errors"
	"fmt"
	"github.com/openshift/openshift-azure/pkg/api"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

var searchPattern = regexp.MustCompile(`(\d+)`)
var replacePattern = []byte(` $1 `)

// padNumbers inserts spaces around numbers within a string
func padNumbers(s string) string {
	b := []byte(s)
	b = searchPattern.ReplaceAll(b, replacePattern)
	return string(b)
}

func TestPadNumbers(t *testing.T) {
	testcases := []struct {
		input    string
		expected string
	}{
		{input: "single123number", expected: "single 123 number"},
		{input: "single123numbers345spread", expected: "single 123 numbers 345 spread"},
		{input: "number_at_end234", expected: "number_at_end 234 "},
		{input: "123number_at_start", expected: " 123 number_at_start"},
		{input: "123number_at_start", expected: " 123 number_at_start"},
	}

	for _, test := range testcases {
		got := padNumbers(test.input)
		if got != test.expected {
			t.Errorf("padNumbers(%v) != %v, expected %v", test.input, got, test.expected)
		}
	}
}

var acronymsPattern = regexp.MustCompile(`HTTP|SKU|ID|SSH|RFC`)
var acronymsReplaceFunc = func(s []byte) []byte { return []byte(strings.Title(strings.ToLower(string(s)))) }

// titlelizeAcronyms converts capitalized acronyms to "Title" format
func titlelizeAcronyms(s string) string {
	b := []byte(s)
	b = acronymsPattern.ReplaceAllFunc(b, acronymsReplaceFunc)
	return string(b)
}

func TestTitlelizeAcronyms(t *testing.T) {
	testcases := []struct {
		input    string
		expected string
	}{
		{input: "HTTPRequest", expected: "HttpRequest"},
		{input: "testSKUConfig", expected: "testSkuConfig"},
		{input: "testID", expected: "testId"},
	}

	for _, test := range testcases {
		got := titlelizeAcronyms(test.input)
		if got != test.expected {
			t.Errorf("titlelizeAcronyms(%v) != %v, expected %v", test.input, got, test.expected)
		}
	}
}

// Converts a string to lower camel case
func lowerCamelCase(s string) string {
	if s == "" {
		return s
	}

	prep := strings.TrimSpace(s)

	// replace known capitalized acronyms with their title
	// e.g. RFC becomes Rfc
	prep = titlelizeAcronyms(prep)

	start, rest := prep[0], prep[1:]
	if start >= 'A' && start <= 'Z' {
		prep = strings.ToLower(string(start)) + rest
	}
	// pad all numbers within the string with spaces
	prep = padNumbers(prep)

	res := ""
	cap := false
	for _, char := range prep {
		if char >= '0' && char <= '9' {
			res += string(char)
		}
		if char >= 'A' && char <= 'Z' {
			res += string(char)
		}
		if char >= 'a' && char <= 'z' {
			if cap {
				res += strings.ToUpper(string(char))
			} else {
				res += string(char)
			}
		}
		// if the current character is special then the
		// next letter encountered should be uppercased
		if char == ' ' || char == '_' || char == '-' {
			cap = true
		} else {
			cap = false
		}
	}

	return res
}

func TestLowerCamelCase(t *testing.T) {

	testcases := []struct {
		input    string
		expected string
	}{
		{input: "", expected: ""},
		{input: "A", expected: "a"},
		{input: "ABUG", expected: "aBUG"},
		{input: "with_underscore", expected: "withUnderscore"},
		{input: "with-hyphen", expected: "withHyphen"},
		{input: "with some-space", expected: "withSomeSpace"},
		{input: "StandardTestCase", expected: "standardTestCase"},
		{input: "ABigDay", expected: "aBigDay"},
		{input: "Some-Numbers_123now", expected: "someNumbers123Now"},
		{input: "Numbers 123with_ spaces", expected: "numbers123WithSpaces"},
		{input: " Spaces at the edges ", expected: "spacesAtTheEdges"},
		{input: " 123 numbers at edges 345 ", expected: "123NumbersAtEdges345"},
	}

	for _, test := range testcases {
		got := lowerCamelCase(test.input)
		if got != test.expected {
			t.Errorf("lowerCamelCase(%v) != %v, expected %v", test.input, got, test.expected)
		}
	}
}

// jstctest represents a json struct tag conformance test
type jstctest struct {
	name     string
	check    func(string) string
	input    interface{}
	expected []error
}

func TestJstLowerCamelCaseCheck(t *testing.T) {
	testcases := []jstctest{
		jstctest{
			name:  "input is nil",
			check: lowerCamelCase,
			input: nil,
			expected: []error{
				errors.New(`cannot perform json struct tags check on "nil"`),
			},
		},
		jstctest{
			name:  "input is a string",
			check: lowerCamelCase,
			input: "test",
			expected: []error{
				errors.New(`cannot perform json struct tags check on "string" type`),
			},
		},
		jstctest{
			name:  "input is a number",
			check: lowerCamelCase,
			input: 2,
			expected: []error{
				errors.New(`cannot perform json struct tags check on "int" type`),
			},
		},
		jstctest{
			name:  "input is a slice",
			check: lowerCamelCase,
			input: []string{"input", "is", "an", "array"},
			expected: []error{
				errors.New(`cannot perform json struct tags check on "[]string" type`),
			},
		},
		jstctest{
			name:  "input is a map",
			check: lowerCamelCase,
			input: make(map[string]string),
			expected: []error{
				errors.New(`cannot perform json struct tags check on "map[string]string" type`),
			},
		},
		jstctest{
			name:  "input is a struct but doesn't specify some json tags",
			check: lowerCamelCase,
			input: struct {
				Name  string `json:"name,omitempty"`
				Value string
			}{
				Name:  "name",
				Value: "value",
			},
			expected: []error{
				errors.New(fmt.Sprintf(`field "Value" does not have a json tag`)),
			},
		},
		jstctest{
			name:  "input is a struct but doesn't specify any json tags",
			check: lowerCamelCase,
			input: struct {
				Name  string
				Value string
			}{},
			expected: []error{
				errors.New(fmt.Sprintf(`field "Name" does not have a json tag`)),
				errors.New(fmt.Sprintf(`field "Value" does not have a json tag`)),
			},
		},
		jstctest{
			name:  "input is a struct with all json tags correctly specified in lower camel case but without extra qualifiers",
			check: lowerCamelCase,
			input: struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			}{},
			expected: []error{},
		},
		jstctest{
			name:  "input is a struct with all json tags correctly specified in lower camel case",
			check: lowerCamelCase,
			input: struct {
				Name        string `json:"name,omitempty"`
				Value       string `json:"value,omitempty"`
				N           string `json:"n,omitempty"`
				Foo_bar     string `json:"fooBar,omitempty"`
				AnotherCase string `json:"anotherCase,omitempty"`
				Rfc123check string `json:"rfc123Check,omitempty"`
			}{},
			expected: []error{},
		},
		jstctest{
			name:  "input is a struct with some json tags not correctly specified in lower camel case",
			check: lowerCamelCase,
			input: struct {
				Name        string `json:"name,omitempty"`
				Value       string `json:"Value,omitempty"`
				N           string `json:"n,omitempty"`
				Foo_bar     string `json:"FooBar,omitempty"`
				AnotherCase string `json:"anotherCase,omitempty"`
			}{},
			expected: []error{
				errors.New(fmt.Sprintf(`field "Value" specifies an incorrect json tag "Value". it should be "value"`)),
				errors.New(fmt.Sprintf(`field "Foo_bar" specifies an incorrect json tag "FooBar". it should be "fooBar"`)),
			},
		},
		jstctest{
			name:     "input is the api.CertKeyPair struct",
			check:    lowerCamelCase,
			input:    api.CertKeyPair{},
			expected: []error{},
		},
		jstctest{
			name:     "input is the api.CertificateConfig struct",
			check:    lowerCamelCase,
			input:    api.CertificateConfig{},
			expected: []error{},
		},
		jstctest{
			name:     "input is the api.ImageConfig struct",
			check:    lowerCamelCase,
			input:    api.ImageConfig{},
			expected: []error{},
		},
		jstctest{
			name:     "input is the api.Config struct",
			check:    lowerCamelCase,
			input:    api.Config{},
			expected: []error{},
		},
	}

	for _, test := range testcases {
		got := jstCheck(test.input, test.check)
		if !reflect.DeepEqual(got, test.expected) {
			t.Errorf(`"%s" testcase expected errors %v but received %v`, test.name, test.expected, got)
		}
	}
}

func jstCheck(o interface{}, check func(s string) string) []error {

	errs := []error{}

	if o == nil {
		errs = append(errs, fmt.Errorf(`cannot perform json struct tags check on "nil"`))
		return errs
	}

	ot := reflect.TypeOf(o)
	if ot.Kind() != reflect.Struct {
		errs = append(errs, fmt.Errorf(`cannot perform json struct tags check on "%v" type`, ot))
		return errs
	}

	ov := reflect.ValueOf(o)
	for i := 0; i < ov.NumField(); i++ {

		field := ov.Type().Field(i)
		tag := field.Tag.Get("json")

		if tag == "" {
			errs = append(errs, fmt.Errorf(`field "%v" does not have a json tag`, field.Name))
			continue
		}

		jtag := ""

		// tags contain the tag name and other qualifiers separated by a comma
		if strings.Contains(tag, ",") {
			jtag = tag[0:strings.Index(tag, ",")]
		} else {
			jtag = tag
		}

		// compute the correct tag name
		ctag := check(field.Name)

		if jtag != ctag {
			errs = append(errs, fmt.Errorf(`field "%v" specifies an incorrect json tag "%v". it should be "%v"`, field.Name, jtag, ctag))
		}
	}
	return errs
}
