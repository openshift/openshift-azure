package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/sync"
)

var (
	dryRun = flag.Bool("n", false, "dry run")
)

func run() error {
	var paths []string

	err := filepath.Walk("pkg/sync/data", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		b1, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		u, err := sync.Unmarshal(b1)
		if err != nil {
			return err
		}

		if err = sync.Clean(u); err != nil {
			log.Print(path)
			return err
		}

		sync.Default(u)

		b2, err := yaml.Marshal(u.Object)
		if err != nil {
			return err
		}

		if *dryRun {
			if !bytes.Equal(b1, b2) {
				paths = append(paths, path)
			}
		} else {
			if err = ioutil.WriteFile(path, b2, 0666); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}
	if *dryRun && len(paths) != 0 {
		return fmt.Errorf("%s wants to change the following files:\n  %s", os.Args[0], strings.Join(paths, "\n  "))
	}

	return nil
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}
