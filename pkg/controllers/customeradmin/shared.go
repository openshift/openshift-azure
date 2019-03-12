package customeradmin

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var desiredRolebindings = map[string]rbacv1.RoleBinding{
	"osa-customer-admin": {
		ObjectMeta: metav1.ObjectMeta{
			Name: "osa-customer-admin",
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Group",
				Name:     "osa-customer-admins",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "admin",
		},
	},
	"osa-customer-admin-project": {
		ObjectMeta: metav1.ObjectMeta{
			Name: "osa-customer-admin-project",
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Group",
				Name:     "osa-customer-admins",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "customer-admin-project",
		},
	},
}

// AddToManager adds all Controllers to the Manager
func AddToManager(ctx context.Context, log *logrus.Entry, m manager.Manager, stopCh <-chan struct{}) error {
	if err := addGroupController(ctx, log, m, stopCh); err != nil {
		return err
	}
	if err := addNamespaceController(log, m); err != nil {
		return err
	}
	if err := addRolebindingController(log, m); err != nil {
		return err
	}
	return nil
}

func ignoredNamespace(namespace string) bool {
	for _, name := range []string{"openshift", "kubernetes", "kube", "default"} {
		if namespace == name {
			return true
		}
	}
	for _, prefix := range []string{"openshift-", "kubernetes-", "kube-"} {
		if strings.HasPrefix(namespace, prefix) {
			return true
		}
	}
	return false
}
