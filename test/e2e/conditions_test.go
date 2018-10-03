//+build e2e

package e2e

import (
	authorizationapiv1 "k8s.io/api/authorization/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
