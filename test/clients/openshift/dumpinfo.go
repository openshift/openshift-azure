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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	artifactDir = flag.String("artifact-dir", "", "Directory to place artifacts when a test fails")
)

// DumpInfo dumps logs and events from the clusters
// to sub-directory in ARTIFACTS folder
func (cli *Client) DumpInfo(namespace, subDir string) error {
	// namespace = "" is same as --all-namespaces
	var dir string
	if os.Getenv("ARTIFACTS") != "" {
		dir = filepath.Join(os.Getenv("ARTIFACTS"), subDir)
	} else {
		dir = filepath.Join(*artifactDir, subDir)
	}
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	if err := cli.dumpEvents(namespace, dir); err != nil {
		return err
	}
	if err := cli.dumpPods(namespace, dir); err != nil {
		return err
	}
	if err := cli.dumpBuilds(namespace, dir); err != nil {
		return err
	}
	if namespace != "" {
		if err := cli.dumpPodsLogs(namespace, dir); err != nil {
			return err
		}
	}
	return nil
}

func (cli *Client) dumpBuildLog(name, namespace, dir string, f *os.File) error {
	buildLogOptions := buildv1client.BuildLogOptions{
		Follow: true,
		NoWait: false,
	}
	var err error
	f, err = os.Create(filepath.Join(dir, fmt.Sprintf("build-%s.log", name)))
	if err != nil {
		logrus.Warn(err)
	} else {
		defer f.Close()
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

func (cli *Client) dumpPodLogs(namespace, dir string) error {

	return nil
}

func (cli *Client) dumpBuilds(namespace, dir string) error {
	out, err := os.OpenFile(filepath.Join(dir, "builds.yaml"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer out.Close()

	list, err := cli.BuildV1.Builds(namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, build := range list.Items {
		b, err := yaml.Marshal(build)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(out, string(b))
		if err != nil {
			return err
		}
		err = cli.dumpBuildLog(build.Name, build.Namespace, dir, os.Stdout)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cli *Client) dumpEvents(namespace, dir string) error {
	out, err := os.OpenFile(filepath.Join(dir, "events.yaml"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer out.Close()

	list, err := cli.CoreV1.Events(namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, item := range list.Items {
		b, err := yaml.Marshal(item)
		if err != nil {
			return err
		}

		if _, err := out.WriteString(string(b)); err != nil {
			return err
		}
	}

	return nil
}

func (cli *Client) dumpPods(namespace, dir string) error {
	out, err := os.OpenFile(filepath.Join(dir, "pods.yaml"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer out.Close()

	list, err := cli.CoreV1.Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, item := range list.Items {
		b, err := yaml.Marshal(item)
		if err != nil {
			return err
		}

		if _, err := out.WriteString(string(b)); err != nil {
			return err
		}
	}

	return nil
}

func (cli *Client) dumpPodsLogs(namespace, dir string) error {
	pods, err := cli.CoreV1.Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	podLogOpt := corev1.PodLogOptions{}
	for _, pod := range pods.Items {
		fmt.Printf("log for %s\n", pod.Name)
		req := cli.CoreV1.Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpt)
		podLogs, err := req.Stream()
		if err != nil {
			return fmt.Errorf("error in opening stream %v", err)
		}
		defer podLogs.Close()

		out, err := os.OpenFile(filepath.Join(dir, pod.Name+".log"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, podLogs)
		if err != nil {
			return fmt.Errorf("error in coping logs %v", err)
		}
	}
	return nil
}
