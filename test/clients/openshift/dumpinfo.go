package openshift

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	buildv1client "github.com/openshift/api/build/v1"
	buildschema "github.com/openshift/client-go/build/clientset/versioned/scheme"
	"github.com/pkg/errors"
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
	if err := cli.dumpBuilds(namespace); err != nil {
		return err
	}
	return nil
}

func (cli *Client) dumpBuildLog(name, namespace string, f *os.File) error {
	buildLogOptions := buildv1client.BuildLogOptions{
		Follow: true,
		NoWait: false,
	}
	if *artifactDir != "" {
		var err error
		f, err = os.Create(filepath.Join(*artifactDir, fmt.Sprintf("build-%s-%s.log", namespace, name)))
		if err != nil {
			logrus.Warn(err)
		} else {
			defer f.Close()
		}
	}

	rd, err := cli.BuildV1.RESTClient().Get().
		Namespace(namespace).
		Resource("builds").
		Name(name).
		SubResource("log").
		VersionedParams(&buildLogOptions, buildschema.ParameterCodec).
		Stream()

	if err != nil {
		return errors.Wrapf(err, "unable get build log  %s/%s", namespace, name)
	}
	defer rd.Close()

	if _, err = io.Copy(f, rd); err != nil {
		return errors.Wrapf(err, "error streaming logs for %s/%s", namespace, name)
	}
	return nil
}

func (cli *Client) dumpBuilds(namespace string) error {
	f := os.Stdout

	if *artifactDir != "" {
		var err error
		f, err = os.Create(filepath.Join(*artifactDir, fmt.Sprintf("builds-%s.yaml", namespace)))
		if err != nil {
			logrus.Warn(err)
		} else {
			defer f.Close()
		}
	}

	list, err := cli.BuildV1.Builds(namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, build := range list.Items {
		b, err := yaml.Marshal(build)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(f, string(b))
		if err != nil {
			return err
		}
		err = cli.dumpBuildLog(build.Name, build.Namespace, os.Stdout)
		if err != nil {
			return err
		}
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
