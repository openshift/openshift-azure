package ready

import (
	"fmt"
	"net"

	csbv1beta1 "github.com/kubernetes-incubator/service-catalog/pkg/apis/servicecatalog/v1beta1"
	csbv1beta1client "github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset/typed/servicecatalog/v1beta1"
	oappsv1 "github.com/openshift/api/apps/v1"
	templatev1 "github.com/openshift/api/template/v1"
	oappsv1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	templatev1client "github.com/openshift/client-go/template/clientset/versioned/typed/template/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	kapiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	kapiextensionsv1beta1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	batchv1client "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	apiregistrationv1client "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1"
)

// APIServiceIsReady returns true if an APIService is considered ready
func APIServiceIsReady(svc *apiregistrationv1.APIService) bool {
	for _, cond := range svc.Status.Conditions {
		if cond.Type == apiregistrationv1.Available &&
			cond.Status == apiregistrationv1.ConditionTrue {
			return true
		}
	}

	return false
}

// CheckAPIServiceIsReady returns a function which polls an APIService and
// returns its readiness
func CheckAPIServiceIsReady(cli apiregistrationv1client.APIServiceInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		svc, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return APIServiceIsReady(svc), nil
	}
}

// CustomResourceDefinitionIsReady returns true if a CustomResourceDefinition is
// considered ready
func CustomResourceDefinitionIsReady(crd *kapiextensionsv1beta1.CustomResourceDefinition) bool {
	for _, cond := range crd.Status.Conditions {
		if cond.Type == kapiextensionsv1beta1.Established &&
			cond.Status == kapiextensionsv1beta1.ConditionTrue {
			return true
		}
	}

	return false
}

// CheckCustomResourceDefinitionIsReady returns a function which polls a
// CustomResourceDefinition and returns its readiness
func CheckCustomResourceDefinitionIsReady(cli kapiextensionsv1beta1client.CustomResourceDefinitionInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		crd, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return CustomResourceDefinitionIsReady(crd), nil
	}
}

// ClusterServiceBrokerIsReady returns true if a ClusterServiceBroker is
// considered ready
func ClusterServiceBrokerIsReady(csb *csbv1beta1.ClusterServiceBroker) bool {
	for _, cond := range csb.Status.Conditions {
		if cond.Status == csbv1beta1.ConditionTrue {
			return true
		}
	}

	return false
}

// CheckClusterServiceBrokerIsReady returns a function which polls a
// CheckClusterServiceBroker and returns its readiness
func CheckClusterServiceBrokerIsReady(cli csbv1beta1client.ClusterServiceBrokerInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		csb, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return ClusterServiceBrokerIsReady(csb), nil
	}
}

// DaemonSetIsReady returns true if a DaemonSet is considered ready
func DaemonSetIsReady(ds *appsv1.DaemonSet) bool {
	return ds.Status.DesiredNumberScheduled == ds.Status.NumberAvailable &&
		ds.Status.DesiredNumberScheduled == ds.Status.UpdatedNumberScheduled &&
		ds.Generation == ds.Status.ObservedGeneration
}

// CheckDaemonSetIsReady returns a function which polls a DaemonSet and returns
// its readiness
func CheckDaemonSetIsReady(cli appsv1client.DaemonSetInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		ds, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return DaemonSetIsReady(ds), nil
	}
}

// DeploymentIsReady returns true if a Deployment is considered ready
func DeploymentIsReady(d *appsv1.Deployment) bool {
	specReplicas := int32(1)
	if d.Spec.Replicas != nil {
		specReplicas = *d.Spec.Replicas
	}

	return specReplicas == d.Status.AvailableReplicas &&
		specReplicas == d.Status.UpdatedReplicas &&
		d.Generation == d.Status.ObservedGeneration
}

// CheckDeploymentIsReady returns a function which polls a Deployment and
// returns its readiness
func CheckDeploymentIsReady(cli appsv1client.DeploymentInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		d, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return DeploymentIsReady(d), nil
	}
}

// DeploymentConfigIsReady returns true if a DeploymentConfig is considered
// ready
func DeploymentConfigIsReady(dc *oappsv1.DeploymentConfig) bool {
	return dc.Spec.Replicas == dc.Status.AvailableReplicas &&
		dc.Spec.Replicas == dc.Status.UpdatedReplicas &&
		dc.Generation == dc.Status.ObservedGeneration

}

// CheckDeploymentConfigIsReady returns a function which polls a
// DeploymentConfig and returns its readiness
func CheckDeploymentConfigIsReady(cli oappsv1client.DeploymentConfigInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		dc, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return DeploymentConfigIsReady(dc), nil
	}
}

// NodeIsReady returns true if a Node is considered ready
func NodeIsReady(node *corev1.Node) bool {
	for _, c := range node.Status.Conditions {
		if c.Type == corev1.NodeReady &&
			c.Status == corev1.ConditionTrue {
			return true
		}
	}

	return false
}

// CheckNodeIsReady returns a function which polls a Node and returns its
// readiness
func CheckNodeIsReady(cli corev1client.NodeInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		node, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return NodeIsReady(node), nil
	}
}

// PodIsReady returns true if a Pod is considered ready
func PodIsReady(pod *corev1.Pod) bool {
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodReady &&
			c.Status == corev1.ConditionTrue {
			return true
		}
	}

	return false
}

// CheckPodIsReady returns a function which polls a Pod and returns its
// readiness
func CheckPodIsReady(cli corev1client.PodInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		pod, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return PodIsReady(pod), nil
	}
}

// PodHasPhase returns true if the phase of a Pod matches the given input
func PodHasPhase(pod *corev1.Pod, phase corev1.PodPhase) bool {
	return pod.Status.Phase == phase
}

// CheckPodHasPhase returns a function which polls a Pod and returns whether its
// phase matches the given input
func CheckPodHasPhase(cli corev1client.PodInterface, name string, phase corev1.PodPhase) func() (bool, error) {
	return func() (bool, error) {
		pod, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return PodHasPhase(pod, phase), nil
	}
}

// JobIsReady returns true if a Job is considered ready
func JobIsReady(job *batchv1.Job) bool {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobComplete &&
			c.Status == corev1.ConditionTrue {
			return true
		}
	}

	return false
}

// CheckJobIsReady returns a function which polls a Job and returns its
// readiness
func CheckJobIsReady(cli batchv1client.JobInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		job, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return JobIsReady(job), nil
	}
}

// StatefulSetIsReady returns true if a StatefulSet is considered ready
func StatefulSetIsReady(ss *appsv1.StatefulSet) bool {
	specReplicas := int32(1)
	if ss.Spec.Replicas != nil {
		specReplicas = *ss.Spec.Replicas
	}

	return specReplicas == ss.Status.ReadyReplicas &&
		specReplicas == ss.Status.UpdatedReplicas &&
		ss.Generation == ss.Status.ObservedGeneration
}

// CheckStatefulSetIsReady returns a function which polls a StatefulSet and
// returns its readiness
func CheckStatefulSetIsReady(cli appsv1client.StatefulSetInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		ss, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return StatefulSetIsReady(ss), nil
	}
}

// TemplateInstanceIsReady returns true if a TemplateInstance is considered ready
func TemplateInstanceIsReady(ti *templatev1.TemplateInstance) (bool, error) {
	for _, cond := range ti.Status.Conditions {
		if cond.Type == templatev1.TemplateInstanceReady &&
			cond.Status == corev1.ConditionTrue {
			return true, nil
		} else if cond.Type == templatev1.TemplateInstanceInstantiateFailure &&
			cond.Status == corev1.ConditionTrue {
			return false, fmt.Errorf("templateinstance %s/%s failed: %s (%s)", ti.Namespace, ti.Name, cond.Reason, cond.Message)
		}
	}

	return false, nil
}

// CheckTemplateInstanceIsReady returns a function which polls a
// TemplateInstance and returns its readiness
func CheckTemplateInstanceIsReady(cli templatev1client.TemplateInstanceInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		ti, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return TemplateInstanceIsReady(ti)
	}
}

// ServiceIsReady returns true if a Service is considered ready
func ServiceIsReady(svc *corev1.Service) (bool, error) {
	switch svc.Spec.Type {
	case corev1.ServiceTypeLoadBalancer:
		return len(svc.Status.LoadBalancer.Ingress) > 0, nil
	case corev1.ServiceTypeClusterIP:
		return net.ParseIP(svc.Spec.ClusterIP) != nil, nil
	default:
		return false, fmt.Errorf("unknown service type")
	}
}

// CheckServiceIsReady returns a function which polls a Service and returns its
// readiness
func CheckServiceIsReady(cli corev1client.ServiceInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		svc, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return ServiceIsReady(svc)
	}
}
