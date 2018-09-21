package api_test

import (
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

// Converts a string to lower camel case
func lowerCamelCase(s string) string {
	if s == "" {
		return s
	}

	prep := strings.TrimSpace(s)
	start, rest := prep[0], prep[1:]
	if start >= 'A' && start <= 'Z' {
		prep = strings.ToLower(string(start)) + rest
	}
	//  pad all numbers within the string with spaces
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
