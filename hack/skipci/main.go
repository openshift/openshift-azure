package main

/*

A list of available environment variables can be found here:

https://github.com/kubernetes/test-infra/blob/a9f967cb1235916fb5a4eca661fa083413211242/prow/jobs.md#job-environment-variables

For periodics:
- JOB_NAME
- JOB_TYPE
- JOB_SPEC
- BUILD_ID
- PROW_JOB_ID

*/

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

var (
	debug = flag.Bool("debug", false, "whether to print debug statements")
)

type commitFile struct {
	extension string
	directory string
	filename  string
	original  string
}

var (
	whiteListDirectories = []string{
		"docs/",
		"hack/skipci",
	}

	whiteListFileExt = []string{
		".md",
		".asciidoc",
	}
	whiteListFile = []string{
		"OWNERS",
		"LICENSE",
		"env.example",
		".codecov.yml",
	}
)

func whiteListed(f commitFile, debug bool) bool {
	for _, wld := range whiteListDirectories {
		if strings.Contains(f.directory, wld) {
			if debug {
				fmt.Printf("matched dir (%s) on whitelist dir (%s) with (%s)\n", f.original, f.directory, wld)
			}
			return true
		}
	}
	for _, wle := range whiteListFileExt {
		if strings.HasSuffix(f.extension, wle) {
			if debug {
				fmt.Printf("matched file (%s) on whitelist file extension (%s) with (%s)\n", f.original, f.extension, wle)
			}
			return true
		}
	}
	for _, wlf := range whiteListFile {
		if strings.EqualFold(f.filename, wlf) {
			if debug {
				fmt.Printf("matched file (%s) on whitelist file (%s) with (%s)\n", f.original, f.filename, wlf)
			}
			return true
		}
	}

	return false
}

func getCommitFiles() ([]commitFile, error) {
	cmd := "git"
	args := []string{"diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD..origin/master"}
	c := exec.Command(cmd, args...)
	cmdOut, err := c.Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}
	files := []commitFile{}
	for _, f := range strings.Split(string(cmdOut), "\n") {
		if f != "" {
			files = append(files, commitFile{
				filename:  path.Base(f),
				extension: path.Ext(f),
				directory: path.Dir(f),
				original:  f,
			})
		}
	}

	return files, nil
}

func main() {
	flag.Parse()
	// algorithm:
	// find all files in recent commits
	// first determine if they are in a directory that is test worthy
	// second determine if they have a file extension that can be ignored
	// any other tests we can think of?
	files, err := getCommitFiles()
	if err != nil {
		panic(err)
	}

	for _, f := range files {
		fmt.Printf("File: %s\n", f.original)
		if !whiteListed(f, *debug) {
			// found a file that requires testing
			// TODO: further classify what testing we should perform
			os.Exit(1)
		}
	}
	os.Exit(0)
}
