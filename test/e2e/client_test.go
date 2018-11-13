//+build e2e

package e2e

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/ghodss/yaml"
	project "github.com/openshift/api/project/v1"
	templatev1 "github.com/openshift/api/template/v1"
	appsv1 "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	projectclient "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	templatev1client "github.com/openshift/client-go/template/clientset/versioned/typed/template/v1"
	userv1client "github.com/openshift/client-go/user/clientset/versioned/typed/user/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var c, cadmin, creader *testClient

type testClient struct {
	ac        *appsv1.AppsV1Client
	kc        *kubernetes.Clientset
	pc        *projectclient.ProjectV1Client
	rc        *routev1client.RouteV1Client
	tc        *templatev1client.TemplateV1Client
	uc        *userv1client.UserV1Client
	namespace string

	artifactDir string
}

func newTestClient(kubeconfig, artifactDir string) *testClient {
	var err error
	var config *rest.Config

	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err)
		}
	} else {
		// use in-cluster config if no kubeconfig has been specified
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	// create the clientset
	kc, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// create a project client for creating and tearing down namespaces
	pc, err := projectclient.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// create a template client
	tc, err := templatev1client.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// create a route client
	rc, err := routev1client.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// create a route client
	uc, err := userv1client.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	ac, err := appsv1.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return &testClient{
		ac:          ac,
		kc:          kc,
		pc:          pc,
		rc:          rc,
		tc:          tc,
		uc:          uc,
		artifactDir: artifactDir,
	}
}

func (t *testClient) createProject(namespace string) error {
	if _, err := t.pc.ProjectRequests().Create(&project.ProjectRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		}}); err != nil {
		return err
	}
	t.namespace = namespace

	if err := wait.PollImmediate(2*time.Second, time.Minute, t.selfSarSuccess); err != nil {
		return fmt.Errorf("failed to wait for self-sar success: %v", err)
	}
	if err := wait.PollImmediate(2*time.Second, time.Minute, t.defaultServiceAccountIsReady); err != nil {
		return fmt.Errorf("failed to wait for the default service account provision: %v", err)
	}
	return nil
}

func (t *testClient) cleanupProject(timeout time.Duration) error {
	if t.namespace == "" {
		return nil
	}
	if err := t.pc.Projects().Delete(t.namespace, &metav1.DeleteOptions{}); err != nil {
		return err
	}
	if err := wait.PollImmediate(2*time.Second, timeout, t.projectIsCleanedUp); err != nil {
		return fmt.Errorf("failed to wait for project cleanup: %v", err)
	}
	return nil
}

func (t *testClient) dumpInfo() error {
	// gather events
	eventList, err := t.kc.CoreV1().Events(t.namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	eventBuf := bytes.NewBuffer(nil)
	for _, event := range eventList.Items {
		b, err := yaml.Marshal(event)
		if err != nil {
			return err
		}
		if _, err := eventBuf.Write(b); err != nil {
			return err
		}
		if _, err := eventBuf.Write([]byte("\n")); err != nil {
			return err
		}
	}

	// gather pods
	podList, err := t.kc.CoreV1().Pods(t.namespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	podBuf := bytes.NewBuffer(nil)
	for _, pod := range podList.Items {
		b, err := yaml.Marshal(pod)
		if err != nil {
			return err
		}
		if _, err := podBuf.Write(b); err != nil {
			return err
		}
		if _, err := podBuf.Write([]byte("\n")); err != nil {
			return err
		}
	}

	if t.artifactDir != "" {
		if err := ioutil.WriteFile(filepath.Join(t.artifactDir, fmt.Sprintf("events-%s.yaml", t.namespace)), eventBuf.Bytes(), 0777); err != nil {
			logrus.Warn(err)
			fmt.Println(eventBuf.String())
		}
		if err := ioutil.WriteFile(filepath.Join(t.artifactDir, fmt.Sprintf("pods-%s.yaml", t.namespace)), podBuf.Bytes(), 0777); err != nil {
			logrus.Warn(err)
			fmt.Println(podBuf.String())
		}
	} else {
		fmt.Println(eventBuf.String())
		fmt.Println(podBuf.String())
	}
	return nil
}

func (t *testClient) instantiateTemplate(tpl string) error {
	// Create the template
	template, err := t.tc.Templates("openshift").Get(
		tpl, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Instantiate the template
	_, err = t.tc.TemplateInstances(c.namespace).Create(
		&templatev1.TemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: c.namespace,
			},
			Spec: templatev1.TemplateInstanceSpec{
				Template: *template,
			},
		})
	if err != nil {
		return err
	}

	// Return after waiting for instance to complete
	return wait.PollImmediate(2*time.Second, 10*time.Minute, c.templateInstanceIsReady)
}

func (t *testClient) loopHTTPGet(url string, regex *regexp.Regexp, times int) func() error {

	httpc := &http.Client{
		Timeout: 10 * time.Second,
	}
	var prevCounter, currCounter int

	return func() error {
		for i := 0; i < times; i++ {
			resp, err := httpc.Get(url)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("unexpected http error returned: %d", resp.StatusCode)
			}

			contents, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			matches := regex.FindStringSubmatch(string(contents))
			if matches == nil {
				return fmt.Errorf("no matches found for %s", regex)
			}

			currCounter, err = strconv.Atoi(matches[1])
			if err != nil {
				return err
			}
			if currCounter <= prevCounter {
				return fmt.Errorf("visit counter didn't increment: %d should be > than %d", currCounter, prevCounter)
			}
			prevCounter = currCounter
			time.Sleep(time.Second)
		}
		return nil
	}
}
