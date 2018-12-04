package ready

import (
	"fmt"

	templatev1 "github.com/openshift/api/template/v1"
	oappsv1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	templatev1client "github.com/openshift/client-go/template/clientset/versioned/typed/template/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	batchv1client "k8s.io/client-go/kubernetes/typed/batch/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	apiregistrationv1client "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1"
)

func APIServiceIsReady(cli apiregistrationv1client.APIServiceInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		svc, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		for _, cond := range svc.Status.Conditions {
			if cond.Type == apiregistrationv1.Available &&
				cond.Status == apiregistrationv1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}
}

func DaemonSetIsReady(cli appsv1client.DaemonSetInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		ds, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		return ds.Status.DesiredNumberScheduled == ds.Status.CurrentNumberScheduled &&
			ds.Status.DesiredNumberScheduled == ds.Status.NumberReady &&
			ds.Status.DesiredNumberScheduled == ds.Status.UpdatedNumberScheduled &&
			ds.Generation == ds.Status.ObservedGeneration, nil
	}
}

func DeploymentIsReady(cli appsv1client.DeploymentInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		d, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		specReplicas := int32(1)
		if d.Spec.Replicas != nil {
			specReplicas = *d.Spec.Replicas
		}

		return specReplicas == d.Status.Replicas &&
			specReplicas == d.Status.ReadyReplicas &&
			specReplicas == d.Status.AvailableReplicas &&
			specReplicas == d.Status.UpdatedReplicas &&
			d.Generation == d.Status.ObservedGeneration, nil
	}
}

func DeploymentConfigIsReady(cli oappsv1client.DeploymentConfigInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		dc, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		specReplicas := dc.Spec.Replicas

		return specReplicas == dc.Status.Replicas &&
			specReplicas == dc.Status.ReadyReplicas &&
			specReplicas == dc.Status.AvailableReplicas &&
			specReplicas == dc.Status.UpdatedReplicas &&
			dc.Generation == dc.Status.ObservedGeneration, nil
	}
}

func NodeIsReady(cli corev1client.NodeInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		node, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		for _, c := range node.Status.Conditions {
			if c.Type == corev1.NodeReady &&
				c.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}
}

func PodIsReady(cli corev1client.PodInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		node, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		for _, c := range node.Status.Conditions {
			if c.Type == corev1.PodReady &&
				c.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}
}

func BatchIsReady(cli batchv1client.JobInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		job, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		for _, c := range job.Status.Conditions {
			if c.Type == batchv1.JobComplete &&
				c.Status == corev1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}
}

func StatefulSetIsReady(cli appsv1client.StatefulSetInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		ss, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		specReplicas := int32(1)
		if ss.Spec.Replicas != nil {
			specReplicas = *ss.Spec.Replicas
		}

		return specReplicas == ss.Status.Replicas &&
			specReplicas == ss.Status.ReadyReplicas &&
			specReplicas == ss.Status.CurrentReplicas &&
			ss.Generation == ss.Status.ObservedGeneration, nil
	}
}

func TemplateInstanceIsReady(cli templatev1client.TemplateInstanceInterface, name string) func() (bool, error) {
	return func() (bool, error) {
		ti, err := cli.Get(name, metav1.GetOptions{})
		switch {
		case errors.IsNotFound(err):
			return false, nil
		case err != nil:
			return false, err
		}

		for _, cond := range ti.Status.Conditions {
			if cond.Type == templatev1.TemplateInstanceReady &&
				cond.Status == corev1.ConditionTrue {
				return true, nil
			} else if cond.Type == templatev1.TemplateInstanceInstantiateFailure &&
				cond.Status == corev1.ConditionTrue {
				return false, fmt.Errorf("templateinstance %s failed", name)
			}
		}

		return false, nil
	}
}
