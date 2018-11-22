//+build e2e

package kubernetes

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/ghodss/yaml"
	dcv1 "github.com/openshift/api/apps/v1"
	project "github.com/openshift/api/project/v1"
	"github.com/openshift/api/route/v1"
	templatev1 "github.com/openshift/api/template/v1"
	usersv1 "github.com/openshift/api/user/v1"
	appsv1 "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	projectclient "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	templatev1client "github.com/openshift/client-go/template/clientset/versioned/typed/template/v1"
	userv1client "github.com/openshift/client-go/user/clientset/versioned/typed/user/v1"
	"github.com/sirupsen/logrus"
	apiappsv1 "k8s.io/api/apps/v1"
	authorizationapiv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/openshift/openshift-azure/pkg/util/log"
	"github.com/openshift/openshift-azure/test/util/client"
)

type Client struct {
	ac        *appsv1.AppsV1Client
	kc        *kubernetes.Clientset
	pc        *projectclient.ProjectV1Client
	rc        *routev1client.RouteV1Client
	tc        *templatev1client.TemplateV1Client
	uc        *userv1client.UserV1Client
	generator *client.SimpleNameGenerator
	namespace string

	artifactDir string
	ctx         context.Context
	logger      *logrus.Entry
}

func NewClient(kubeconfig, artifactDir string) *Client {
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

	ctx := context.Background()

	logrus.SetLevel(log.SanitizeLogLevel("Debug"))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logger := logrus.WithFields(nil)

	return &Client{
		ac:          ac,
		kc:          kc,
		pc:          pc,
		rc:          rc,
		tc:          tc,
		uc:          uc,
		generator:   &client.SimpleNameGenerator{},
		artifactDir: artifactDir,
		logger:      logger,
		ctx:         ctx,
	}
}

func (t *Client) CreateProject(namespace string) error {
	if _, err := t.pc.ProjectRequests().Create(&project.ProjectRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		}}); err != nil {
		return err
	}
	t.namespace = namespace

	if err := wait.PollImmediate(2*time.Second, time.Minute, t.SelfSarSuccess); err != nil {
		return fmt.Errorf("failed to wait for self-sar success: %v", err)
	}
	if err := wait.PollImmediate(2*time.Second, time.Minute, t.DefaultServiceAccountIsReady); err != nil {
		return fmt.Errorf("failed to wait for the default service account provision: %v", err)
	}
	return nil
}

func (t *Client) GenerateRandomName(prefix string) string {
	return t.generator.Generate(prefix)
}

func (t *Client) CleanupProject(timeout time.Duration) error {
	if t.namespace == "" {
		return nil
	}
	if err := t.pc.Projects().Delete(t.namespace, &metav1.DeleteOptions{}); err != nil {
		return err
	}
	if err := wait.PollImmediate(2*time.Second, timeout, t.ProjectIsCleanedUp); err != nil {
		return fmt.Errorf("failed to wait for project cleanup: %v", err)
	}
	return nil
}

func (t *Client) DumpInfo() error {
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

func (t *Client) InstantiateTemplate(tpl string) error {
	// Create the template
	template, err := t.tc.Templates("openshift").Get(
		tpl, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Instantiate the template
	_, err = t.tc.TemplateInstances(t.namespace).Create(
		&templatev1.TemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: t.namespace,
			},
			Spec: templatev1.TemplateInstanceSpec{
				Template: *template,
			},
		})
	if err != nil {
		return err
	}

	// Return after waiting for instance to complete
	return wait.PollImmediate(2*time.Second, 10*time.Minute, t.TemplateInstanceIsReady)
}

func (t *Client) TemplateInstanceIsReady() (bool, error) {
	ti, err := t.tc.TemplateInstances(t.namespace).Get(t.namespace, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	for _, cond := range ti.Status.Conditions {
		if cond.Type == templatev1.TemplateInstanceReady &&
			cond.Status == corev1.ConditionTrue {
			return true, nil
		} else if cond.Type == templatev1.TemplateInstanceInstantiateFailure &&
			cond.Status == corev1.ConditionTrue {
			return false, fmt.Errorf("templateinstance %q failed", t.namespace)
		}
	}
	return false, nil
}

func (t *Client) DeploymentConfigIsReady(name string, replicas int32) func() (bool, error) {
	return func() (bool, error) {
		dc, err := t.ac.DeploymentConfigs(t.namespace).Get(name, metav1.GetOptions{})
		switch {
		case err == nil:
			return replicas == dc.Status.Replicas &&
				replicas == dc.Status.ReadyReplicas &&
				replicas == dc.Status.AvailableReplicas &&
				replicas == dc.Status.UpdatedReplicas &&
				dc.Generation == dc.Status.ObservedGeneration, nil
		default:
			return false, err
		}
	}
}

func (t *Client) ProjectIsCleanedUp() (bool, error) {
	_, err := t.pc.Projects().Get(t.namespace, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return true, nil
	}
	return false, err
}

func (t *Client) DefaultServiceAccountIsReady() (bool, error) {
	sa, err := t.kc.CoreV1().ServiceAccounts(t.namespace).Get("default", metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return len(sa.Secrets) > 0, nil
}

func (t *Client) SelfSarSuccess() (bool, error) {
	res, err := t.kc.AuthorizationV1().SelfSubjectAccessReviews().Create(
		&authorizationapiv1.SelfSubjectAccessReview{
			Spec: authorizationapiv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authorizationapiv1.ResourceAttributes{
					Namespace: t.namespace,
					Verb:      "create",
					Resource:  "pods",
				},
			},
		},
	)
	if err != nil {
		return false, err
	}
	return res.Status.Allowed, nil
}

func (t *Client) CreatePodDisruptionBudget(p *policy.PodDisruptionBudget) error {
	_, err := t.kc.PolicyV1beta1().PodDisruptionBudgets(t.namespace).Create(p)
	return err
}

func (t *Client) GetRoute(template string, options *metav1.GetOptions) (*v1.Route, error) {
	if options == nil {
		options = &metav1.GetOptions{}
	}
	return t.rc.Routes(t.namespace).Get(template, *options)
}

func (t *Client) ListSecrets(namespace string, options *metav1.ListOptions) (*corev1.SecretList, error) {
	if options == nil {
		options = &metav1.ListOptions{}
	}
	return t.kc.CoreV1().Secrets(namespace).List(*options)
}

func (t *Client) ListPods(namespace string, options *metav1.ListOptions) (*corev1.PodList, error) {
	if options == nil {
		options = &metav1.ListOptions{}
	}
	return t.kc.CoreV1().Pods(namespace).List(*options)
}

func (t *Client) GetPodByName(namespace, name string, options *metav1.GetOptions) (*corev1.Pod, error) {
	if options == nil {
		options = &metav1.GetOptions{}
	}
	return t.kc.CoreV1().Pods(namespace).Get(name, *options)
}

func (t *Client) DeletePod(namespace, name string, options *metav1.DeleteOptions) error {
	if options == nil {
		options = &metav1.DeleteOptions{}
	}
	err := t.kc.CoreV1().Pods(namespace).Delete(name, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return wait.PollImmediate(2*time.Second, 1*time.Minute, func() (bool, error) {
		_, err := t.kc.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		return false, nil
	})
}

func (t *Client) GetPodLogs(namespace string, name string, options *corev1.PodLogOptions) *rest.Request {
	if options == nil {
		options = &corev1.PodLogOptions{}
	}
	return t.kc.CoreV1().Pods(namespace).GetLogs(name, options)
}

func (t *Client) ListNodes(options *metav1.ListOptions) (*corev1.NodeList, error) {
	if options == nil {
		options = &metav1.ListOptions{}
	}
	return t.kc.CoreV1().Nodes().List(*options)
}

func (t *Client) GetNode(name string, options *metav1.GetOptions) (*corev1.Node, error) {
	if options == nil {
		options = &metav1.GetOptions{}
	}
	return t.kc.CoreV1().Nodes().Get(name, *options)
}

func (t *Client) GetGroup(name string, options *metav1.GetOptions) (*usersv1.Group, error) {
	if options == nil {
		options = &metav1.GetOptions{}
	}
	return t.uc.Groups().Get(name, metav1.GetOptions{})
}

func (t *Client) GetProject(namespace string, options *metav1.GetOptions) (*project.Project, error) {
	if options == nil {
		options = &metav1.GetOptions{}
	}
	return t.pc.Projects().Get(namespace, *options)
}

func (t *Client) DeleteProject(namespace string, options *metav1.DeleteOptions) error {
	return t.pc.Projects().Delete(namespace, options)
}

func (t *Client) GetNamespace(name string, options *metav1.GetOptions) (*corev1.Namespace, error) {
	if options == nil {
		options = &metav1.GetOptions{}
	}
	return t.kc.CoreV1().Namespaces().Get(name, *options)
}

func (t *Client) ListConfigMaps(namespace string, options *metav1.ListOptions) (*corev1.ConfigMapList, error) {
	if options == nil {
		options = &metav1.ListOptions{}
	}
	return t.kc.CoreV1().ConfigMaps(namespace).List(*options)
}

func (t *Client) GetStatefulSet(namespace, name string, options *metav1.GetOptions) (*apiappsv1.StatefulSet, error) {
	if options == nil {
		options = &metav1.GetOptions{}
	}
	return t.kc.AppsV1().StatefulSets("openshift-monitoring").Get("prometheus-k8s", metav1.GetOptions{})
}

func (t *Client) GetRoleBinding(namespace, name string, options *metav1.GetOptions) (*rbacv1.RoleBinding, error) {
	if options == nil {
		options = &metav1.GetOptions{}
	}
	return t.kc.RbacV1().RoleBindings(namespace).Get(name, *options)
}

func (t *Client) CreateClusterRoleBinding(roleBinding *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
	return t.kc.RbacV1().ClusterRoleBindings().Create(roleBinding)
}

func (t *Client) DeleteClusterRoleBinding(name string, options *metav1.DeleteOptions) error {
	if options == nil {
		options = &metav1.DeleteOptions{}
	}
	return t.kc.RbacV1().ClusterRoleBindings().Delete(name, options)
}

func (t *Client) DeleteClusterRole(name string, options *metav1.DeleteOptions) error {
	if options == nil {
		options = &metav1.DeleteOptions{}
	}
	return t.kc.RbacV1().ClusterRoles().Delete(name, options)
}

func (t *Client) GetDeploymentConfig(name string, options *metav1.GetOptions) (*dcv1.DeploymentConfig, error) {
	if options == nil {
		options = &metav1.GetOptions{}
	}
	return t.ac.DeploymentConfigs(t.namespace).Get(name, *options)
}

func (t *Client) UpdateDeploymentConfig(dc *dcv1.DeploymentConfig) (*dcv1.DeploymentConfig, error) {
	return t.ac.DeploymentConfigs(t.namespace).Update(dc)
}

func (t *Client) CreateSampleDeployment(namespace, name string, replicas int) error {
	int32Ptr := func(i int32) *int32 { return &i }
	deployment := &apiappsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: apiappsv1.DeploymentSpec{
			Replicas: int32Ptr(int32(replicas)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
								{
									Weight: 100,
									PodAffinityTerm: corev1.PodAffinityTerm{
										TopologyKey: "kubernetes.io/hostname",
										LabelSelector: &metav1.LabelSelector{
											MatchExpressions: []metav1.LabelSelectorRequirement{
												{
													Key:      "app",
													Operator: metav1.LabelSelectorOperator("In"),
													Values:   []string{name},
												},
											},
										},
									},
								},
							},
						},
					},
					NodeSelector: map[string]string{
						"node-role.kubernetes.io/compute": "true",
					},
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: "openshift/hello-openshift",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}
	_, err := t.kc.AppsV1().Deployments(namespace).Create(deployment)
	if err != nil {
		return err
	}
	return wait.PollImmediate(2*time.Second, 10*time.Minute, func() (bool, error) {
		d, err := t.kc.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}

		for _, cond := range d.Status.Conditions {
			if cond.Type == "Available" &&
				cond.Status == corev1.ConditionTrue {
				return true, nil
			} else if cond.Type == "ReplicaFailure" &&
				cond.Status == corev1.ConditionTrue {
				return false, fmt.Errorf("deployment %s.%s failed", namespace, name)
			}
		}
		return false, nil
	})
}

func (t *Client) GetDeployment(namespace, name string) (*apiappsv1.Deployment, error) {
	return t.kc.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
}

func (t *Client) DeleteDeployment(namespace, name string) error {
	return t.kc.AppsV1().Deployments(namespace).Delete(name, &metav1.DeleteOptions{})
}

func (t *Client) GetDeploymentPods(namespace, name string) ([]corev1.Pod, error) {
	dep, err := t.GetDeployment(namespace, name)
	if err != nil {
		return nil, err
	}
	set := labels.Set(dep.Spec.Template.Labels)
	listOptions := metav1.ListOptions{LabelSelector: set.AsSelector().String()}
	pods, err := t.ListPods(namespace, &listOptions)
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

func (t *Client) DeleteDeploymentPods(namespace, name string) error {
	pods, err := t.GetDeploymentPods(namespace, name)
	if err != nil {
		return err
	}
	for _, pod := range pods {
		err := t.DeletePod(pod.Namespace, pod.Name, nil)
		if err != nil {
			return err
		}
	}
	return nil
}
