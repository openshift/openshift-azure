package openshift

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	artifactDir = flag.String("artifact-dir", "", "Directory to place artifacts when a test fails")
)

func (cli *Client) DumpInfo(namespace string) error {
	if err := cli.dumpEvents(namespace); err != nil {
		return err
	}
	if err := cli.dumpPods(namespace); err != nil {
		return err
	}
	return nil
}

func (cli *Client) dumpEvents(namespace string) error {
	f := os.Stdout

	if *artifactDir != "" {
		var err error
		f, err = os.Create(filepath.Join(*artifactDir, fmt.Sprintf("events-%s.yaml", namespace)))
		if err != nil {
			logrus.Warn(err)
		} else {
			defer f.Close()
		}
	}

	list, err := cli.CoreV1.Events(namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, event := range list.Items {
		b, err := yaml.Marshal(event)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(f, string(b))
		if err != nil {
			return err
		}
	}

	return nil
}

func (cli *Client) dumpPods(namespace string) error {
	f := os.Stdout

	if *artifactDir != "" {
		var err error
		f, err = os.Create(filepath.Join(*artifactDir, fmt.Sprintf("pods-%s.yaml", namespace)))
		if err != nil {
			logrus.Warn(err)
		} else {
			defer f.Close()
		}
	}

	list, err := cli.CoreV1.Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, event := range list.Items {
		b, err := yaml.Marshal(event)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(f, string(b))
		if err != nil {
			return err
		}
	}

	return nil
}
