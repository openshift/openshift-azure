package main

import (
	"testing"
)

func TestDummyCodeCov(t *testing.T) {
	tests := []struct {
		name   string
		cf     commitFile
		status bool
	}{
		{
			name: "markdown",
			cf: commitFile{
				extension: ".md",
				directory: "docs",
				filename:  "test",
				original:  "docs/test.md",
			},
			status: true,
		},
		{
			name: "asciidoc",
			cf: commitFile{
				extension: ".asciidoc",
				directory: "pkg/arm",
				filename:  "test",
				original:  "docs/test.asciidoc",
			},
			status: true,
		},
		{
			name: "gofile",
			cf: commitFile{
				extension: ".go",
				directory: "pkg/startup",
				filename:  "startup",
				original:  "pkg/startup/startup.go",
			},
			status: false,
		},
		{
			name: "shellfile1",
			cf: commitFile{
				extension: ".sh",
				directory: "hack/test",
				filename:  "create",
				original:  "hack/test/create.sh",
			},
			status: false,
		},
		{
			name: "owners",
			cf: commitFile{
				extension: "",
				directory: "",
				filename:  "OWNERS",
				original:  "OWNERS",
			},
			status: true,
		},
	}

	for _, tt := range tests {
		result := whiteListed(tt.cf, false)
		if result != tt.status {
			t.Errorf("%s test failed.  expected %v got=%v", tt.name, tt.status, result)
		}
	}
}
