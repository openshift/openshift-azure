package standard

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"

	"github.com/Azure/go-autorest/autorest/to"
	apiappsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/util/ready"
)

func (sc *SanityChecker) checkMonitoringStackHealth(ctx context.Context) error {
	err := wait.Poll(2*time.Second, 20*time.Minute, ready.CheckDeploymentIsReady(sc.Client.Admin.AppsV1.Deployments("openshift-monitoring"), "cluster-monitoring-operator"))
	if err != nil {
		return err
	}
	err = wait.Poll(2*time.Second, 20*time.Minute, ready.CheckDeploymentIsReady(sc.Client.Admin.AppsV1.Deployments("openshift-monitoring"), "prometheus-operator"))
	if err != nil {
		return err
	}
	err = wait.Poll(2*time.Second, 20*time.Minute, ready.CheckDeploymentIsReady(sc.Client.Admin.AppsV1.Deployments("openshift-monitoring"), "grafana"))
	if err != nil {
		return err
	}
	err = wait.Poll(2*time.Second, 20*time.Minute, ready.CheckStatefulSetIsReady(sc.Client.Admin.AppsV1.StatefulSets("openshift-monitoring"), "prometheus-k8s"))
	if err != nil {
		return err
	}
	err = wait.Poll(2*time.Second, 20*time.Minute, ready.CheckStatefulSetIsReady(sc.Client.Admin.AppsV1.StatefulSets("openshift-monitoring"), "alertmanager-main"))
	if err != nil {
		return err
	}
	err = wait.Poll(2*time.Second, 20*time.Minute, ready.CheckDaemonSetIsReady(sc.Client.Admin.AppsV1.DaemonSets("openshift-monitoring"), "node-exporter"))
	if err != nil {
		return err
	}
	err = wait.Poll(2*time.Second, 20*time.Minute, ready.CheckDeploymentIsReady(sc.Client.Admin.AppsV1.Deployments("openshift-azure-monitoring"), "metrics-bridge"))
	if err != nil {
		return err
	}
	return nil
}

func (sc *SanityChecker) checkNodesLabelledCorrectly(ctx context.Context) error {
	labels := map[string]map[string]string{
		"master": {
			"node-role.kubernetes.io/master": "true",
			"openshift-infra":                "apiserver",
		},
		"compute": {
			"node-role.kubernetes.io/compute": "true",
		},
		"infra": {
			"node-role.kubernetes.io/infra": "true",
		},
	}
	list, err := sc.Client.Admin.CoreV1.Nodes().List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, node := range list.Items {
		kind := strings.Split(node.Name, "-")[0]
		if _, ok := labels[kind]; !ok {
			return fmt.Errorf("map does not have key %s", kind)
		}
		for k, v := range labels[kind] {
			if val, ok := node.Labels[k]; !ok || val != v {
				return fmt.Errorf("map does not have key %s", kind)
			}
		}
	}
	return nil
}

func (sc *SanityChecker) checkDisallowsPdbMutations(ctx context.Context) error {
	namespace, err := sc.createProject(ctx)
	if err != nil {
		return err
	}
	defer sc.deleteProject(ctx, namespace)

	maxUnavailable := intstr.FromInt(1)
	selector, err := metav1.ParseToLabelSelector("key=value")
	if err != nil {
		return err
	}
	pdb := &policy.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: policy.PodDisruptionBudgetSpec{
			MaxUnavailable: &maxUnavailable,
			Selector:       selector,
		},
	}
	_, err = sc.Client.EndUser.PolicyV1beta1.PodDisruptionBudgets(namespace).Create(pdb)
	if kerrors.IsForbidden(err) != true {
		return err
	}
	return nil
}

func (sc *SanityChecker) checkCannotAccessInfraResources(ctx context.Context) error {
	// attempt to read secrets
	_, err := sc.Client.EndUser.CoreV1.Secrets("default").List(metav1.ListOptions{})
	if kerrors.IsForbidden(err) != true {
		return err
	}

	// attempt to list pods
	_, err = sc.Client.EndUser.CoreV1.Pods("default").List(metav1.ListOptions{})
	if kerrors.IsForbidden(err) != true {
		return err
	}

	// attempt to fetch pod by name
	_, err = sc.Client.EndUser.CoreV1.Pods("kube-system").Get("api-master-000000", metav1.GetOptions{})
	if kerrors.IsForbidden(err) != true {
		return err
	}

	// attempt to escalate privileges
	_, err = sc.Client.EndUser.RbacV1.ClusterRoleBindings().Create(&rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-escalate-cluster-admin",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "User",
				Name: "enduser",
			},
		},
		RoleRef: rbacv1.RoleRef{
			Name: "cluster-admin",
			Kind: "ClusterRole",
		},
	})
	if kerrors.IsForbidden(err) != true {
		return err
	}

	// attempt to delete clusterrolebindings
	err = sc.Client.EndUser.RbacV1.ClusterRoleBindings().Delete("cluster-admin", &metav1.DeleteOptions{})
	if kerrors.IsForbidden(err) != true {
		return err
	}

	// attempt to delete clusterrole
	err = sc.Client.EndUser.RbacV1.ClusterRoles().Delete("cluster-admin", &metav1.DeleteOptions{})
	if kerrors.IsForbidden(err) != true {
		return err
	}

	// attempt to fetch pod logs
	req := sc.Client.EndUser.CoreV1.Pods("kube-system").GetLogs("sync-master-000000", &v1.PodLogOptions{})
	result := req.Do()
	errmsg := result.Error().Error()
	expected := "pods \"sync-master-000000\" is forbidden: User \"enduser\" cannot get pods/log in the namespace \"kube-system\""
	if !strings.Contains(errmsg, expected) {
		return fmt.Errorf("could not find expected string in error message [expected: %s, msg: %s]", expected, errmsg)
	}
	return nil
}

func (sc *SanityChecker) checkCanDeployRedhatIoImages(ctx context.Context) error {
	namespace, err := sc.createProject(ctx)
	if err != nil {
		return err
	}
	defer sc.deleteProject(ctx, namespace)

	// nginx 1.14 is in private registry only (so far)
	deploymentName := "redis-32-rhel7"
	privateImage := fmt.Sprintf("registry.redhat.io/rhscl/%s", deploymentName)
	By(fmt.Sprintf("building deployment spec for %s (%v)", privateImage, time.Now()))
	deployment := &apiappsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: namespace,
		},
		Spec: apiappsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": deploymentName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: deploymentName,
					Labels: map[string]string{
						"app": deploymentName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  deploymentName,
							Image: privateImage,
						},
					},
				},
			},
		},
	}
	By(fmt.Sprintf("creating deployment (%v)", time.Now()))
	_, err = sc.Client.EndUser.AppsV1.Deployments(namespace).Create(deployment)
	if err != nil {
		return err
	}
	By(fmt.Sprintf("waiting for deployment to be ready (%v)", time.Now()))
	err = wait.PollImmediate(2*time.Second, 5*time.Minute, ready.CheckDeploymentIsReady(sc.Client.EndUser.AppsV1.Deployments(namespace), deploymentName))
	if err != nil {
		return err
	}
	return nil
}

func (sc *SanityChecker) checkCanCreateLB(ctx context.Context) error {
	namespace, err := sc.createProject(ctx)
	if err != nil {
		return err
	}
	defer sc.deleteProject(ctx, namespace)

	// create standard external loadbalancer
	err = sc.createService("elb", namespace, corev1.ServiceTypeLoadBalancer, map[string]string{})
	if err != nil {
		return err
	}
	// create azure internal loadbalancer
	err = sc.createService("ilb", namespace, corev1.ServiceTypeLoadBalancer, map[string]string{
		"service.beta.kubernetes.io/azure-load-balancer-internal": "true",
	})
	if err != nil {
		return err
	}

	for _, lb := range []string{"elb", "ilb"} {
		By(fmt.Sprintf("waiting for %s to be ready (%v)", lb, time.Now()))
		err = wait.PollImmediate(2*time.Second, 5*time.Minute, ready.CheckServiceIsReady(sc.Client.EndUser.CoreV1.Services(namespace), lb))
		if err != nil {
			return err
		}
	}

	return nil
}

func (sc *SanityChecker) checkCanAccessServices(ctx context.Context) error {
	for _, svc := range []struct {
		url  string
		cert *x509.Certificate
	}{
		{
			url:  "https://" + sc.cs.Properties.PublicHostname + "/healthz",
			cert: sc.cs.Config.Certificates.OpenShiftConsole.Certs[len(sc.cs.Config.Certificates.OpenShiftConsole.Certs)-1],
		},
		{
			url:  "https://console." + sc.cs.Properties.RouterProfiles[0].PublicSubdomain + "/health",
			cert: sc.cs.Config.Certificates.Router.Certs[len(sc.cs.Config.Certificates.Router.Certs)-1],
		},
		{
			url:  "https://docker-registry." + sc.cs.Properties.RouterProfiles[0].PublicSubdomain + "/healthz",
			cert: sc.cs.Config.Certificates.Router.Certs[len(sc.cs.Config.Certificates.Router.Certs)-1],
		},
		{
			url:  "https://registry-console." + sc.cs.Properties.RouterProfiles[0].PublicSubdomain + "/ping",
			cert: sc.cs.Config.Certificates.Router.Certs[len(sc.cs.Config.Certificates.Router.Certs)-1],
		},
	} {
		pool := x509.NewCertPool()
		pool.AddCert(svc.cert)

		cli := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: pool,
				},
			},
			Timeout: 10 * time.Second,
		}

		By(fmt.Sprintf("checking %s", svc.url))
		resp, err := cli.Get(svc.url)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code %d", resp.StatusCode)
		}
	}

	return nil
}

func (sc *SanityChecker) checkCanUseAzureFileStorage(ctx context.Context) error {
	namespace, err := sc.createProject(ctx)
	if err != nil {
		return err
	}
	defer sc.deleteProject(ctx, namespace)

	pvcStorage, err := resource.ParseQuantity("2Gi")
	if err != nil {
		return err
	}

	pvcName := "azure-file"
	storageClass := "azure-file"
	By(fmt.Sprintf("Creating PVC %s in namespace %s", pvcName, namespace))
	_, err = sc.Client.EndUser.CoreV1.PersistentVolumeClaims(namespace).Create(&corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvcName,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.PersistentVolumeAccessMode("ReadWriteMany"),
			},
			StorageClassName: to.StringPtr(storageClass),
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: pvcStorage,
				},
			},
		},
	})
	if err != nil {
		return err
	}
	By(fmt.Sprintf("Created PVC %s", pvcName))

	podName := "busybox"
	By("Creating a simple pod to run dd")
	_, err = sc.Client.EndUser.CoreV1.Pods(namespace).Create(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  podName,
					Image: podName,
					Command: []string{
						"/bin/dd",
						"if=/dev/urandom",
						fmt.Sprintf("of=/data/%s.bin", namespace),
						"bs=1M",
						"count=100",
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      fmt.Sprintf("%s-vol", pvcName),
							MountPath: "/data",
							ReadOnly:  false,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: fmt.Sprintf("%s-vol", pvcName),
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	})
	if err != nil {
		return err
	}
	By("Created pod")
	By(fmt.Sprintf("Waiting for pod %s to finish", podName))
	err = wait.PollImmediate(2*time.Second, 10*time.Minute, ready.CheckPodHasPhase(sc.Client.Admin.CoreV1.Pods(namespace), podName, corev1.PodSucceeded))
	if err != nil {
		return err
	}
	By(fmt.Sprintf("Pod %s finished", podName))

	return nil
}
