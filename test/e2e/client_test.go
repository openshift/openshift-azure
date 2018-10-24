//+build e2e

package e2e

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	project "github.com/openshift/api/project/v1"
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
	"k8s.io/client-go/tools/clientcmd/api"
)

type Config struct {
	// Directory to place artifacts when a test fails
	ArtifactDir string `envconfig:"ARTIFACT_DIR"`

	// Location of the kubeconfig
	KubeConfig string `envconfig:"KUBECONFIG" required:"true"`
}

var c, cadmin, creader *testClient

type testClient struct {
	kc        *kubernetes.Clientset
	pc        *projectclient.ProjectV1Client
	rc        *routev1client.RouteV1Client
	tc        *templatev1client.TemplateV1Client
	uc        *userv1client.UserV1Client
	namespace string

	artifactDir       string
	kubeConfigContext string
}

func newTestClient(kubeconfig, kubeConfigContext, artifactDir string) *testClient {
	var err error
	var restConfig *rest.Config

	if kubeconfig != "" {
		configOptions := clientcmd.NewDefaultPathOptions()
		mergedConfig, err := mergedKubeConfig(configOptions)
		if err != nil {
			panic(err)
		}
		configOverride := &clientcmd.ConfigOverrides{
			CurrentContext: kubeConfigContext,
		}
		restConfig, err = clientcmd.NewDefaultClientConfig(*mergedConfig, configOverride).ClientConfig()
		if err != nil {
			panic(err)
		}
	} else {
		// use in-cluster config if no kubeconfig has been specified
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	// create the clientset
	kc, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		panic(err)
	}

	// create a project client for creating and tearing down namespaces
	pc, err := projectclient.NewForConfig(restConfig)
	if err != nil {
		panic(err)
	}

	// create a template client
	tc, err := templatev1client.NewForConfig(restConfig)
	if err != nil {
		panic(err)
	}

	// create a route client
	rc, err := routev1client.NewForConfig(restConfig)
	if err != nil {
		panic(err)
	}

	// create a route client
	uc, err := userv1client.NewForConfig(restConfig)
	if err != nil {
		panic(err)
	}

	return &testClient{
		kc:                kc,
		pc:                pc,
		rc:                rc,
		tc:                tc,
		uc:                uc,
		artifactDir:       artifactDir,
		kubeConfigContext: kubeConfigContext,
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

// inFocus checks if the supplied focus is part of the test suite description
func inFocus(suiteDescription, focus string) bool {
	modfocus := fmt.Sprintf("\\[%s\\]", focus)
	return strings.Contains(suiteDescription, modfocus)
}

// mergedKubeConfig returns the merged kube config taking into account the cases of single files and a list of files
// provided in the format KUBECONFIG=file-a:file-b:file-c. This is the same functionality used to merged kube configs
// in (kubectl | oc) config view
func mergedKubeConfig(configOptions clientcmd.ConfigAccess) (*api.Config, error) {
	if configOptions.IsExplicitFile() {
		conf, err := clientcmd.LoadFromFile(configOptions.GetExplicitFile())
		if err != nil {
			return nil, err
		}
		return conf, nil
	} else {
		conf, err := configOptions.GetStartingConfig()
		if err != nil {
			return nil, err
		}
		return conf, nil
	}
}
