//+build e2e

package e2e

import (
	"fmt"

	templatev1 "github.com/openshift/api/template/v1"
	authorizationapiv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (t *testClient) templateInstanceIsReady() (bool, error) {
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

func (t *testClient) deploymentConfigIsReady(name string, replicas int32) func() (bool, error) {
	return func() (bool, error) {
		dc, err := c.ac.DeploymentConfigs(c.namespace).Get(name, metav1.GetOptions{})
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

func (t *testClient) projectIsCleanedUp() (bool, error) {
	_, err := t.pc.Projects().Get(t.namespace, metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return true, nil
	}
	return false, err
}

func (t *testClient) defaultServiceAccountIsReady() (bool, error) {
	sa, err := t.kc.CoreV1().ServiceAccounts(t.namespace).Get("default", metav1.GetOptions{})
	if kerrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return len(sa.Secrets) > 0, nil
}

func (t *testClient) selfSarSuccess() (bool, error) {
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
