package sync

import (
	"reflect"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/openshift/openshift-azure/pkg/cluster/kubeclient"
	"github.com/openshift/openshift-azure/pkg/util/jsonpath"
)

// isDouble indicates if we should ignore a given GroupKind because it is
// accessible via a different API route.
func isDouble(gk schema.GroupKind) bool {
	switch gk.String() {
	case "ClusterRole.authorization.openshift.io", // ClusterRole.rbac.authorization.k8s.io
		"ClusterRoleBinding.authorization.openshift.io", // ClusterRoleBinding.rbac.authorization.k8s.io
		"Role.authorization.openshift.io",               // Role.rbac.authorization.k8s.io
		"RoleBinding.authorization.openshift.io",        // RoleBinding.rbac.authorization.k8s.io
		"DaemonSet.extensions",                          // DaemonSet.apps
		"Deployment.extensions",                         // Deployment.apps
		"ImageStreamTag.image.openshift.io",             // ImageStream.image.openshift.io
		"ReplicaSet.extensions",                         // ReplicaSet.apps
		"Project.project.openshift.io",                  // Namespace
		"SecurityContextConstraints":                    // SecurityContextConstraints.security.openshift.io
		return true
	}

	return false
}

// Wants determines if we want to handle the object.
func wants(o unstructured.Unstructured) bool {
	gk := o.GroupVersionKind().GroupKind()
	ns := o.GetNamespace()

	if isDouble(gk) {
		return false
	}

	// skip these API groups.
	switch gk.Group {
	case "events.k8s.io",
		"network.openshift.io",
		"user.openshift.io":
		return false
	}

	// skip non-infrastructure namespaces.
	if !kubeclient.IsControlPlaneNamespace(ns) {
		return false
	}

	switch gk.String() {
	// skip these group kinds.
	case "CertificateSigningRequest.certificates.k8s.io",
		"ClusterServiceClass.servicecatalog.k8s.io",
		"ClusterServicePlan.servicecatalog.k8s.io",
		"ComponentStatus",
		"ControllerRevision.apps",
		"Endpoints",
		"Event",
		"Image.image.openshift.io",
		"ImageStreamTag.image.openshift.io",
		"Node",
		"OAuthAccessToken.oauth.openshift.io",
		"RangeAllocation.security.openshift.io":
		return false

	case "APIService.apiregistration.k8s.io":
		// TODO: don't know exactly what we should do here.
		if _, found := o.GetLabels()["kube-aggregator.kubernetes.io/automanaged"]; found {
			return false
		}

	case "ConfigMap":
		if _, found := o.GetAnnotations()["control-plane.alpha.kubernetes.io/leader"]; found {
			return false
		}

	case "Namespace":
		// skip non-infrastructure namespaces.
		if !kubeclient.IsControlPlaneNamespace(ns) {
			return false
		}

	case "OAuthClient.oauth.openshift.io":
		// TODO: don't know exactly what we should do here.
		switch o.GetName() {
		case "openshift-browser-client",
			"openshift-challenging-client",
			"openshift-web-console":
			return false
		}

	case "Pod":
		// Azure: don't export the etcd, master API, master controllers pods.
		if ns == "kube-system" {
			return false
		}

		for _, ref := range o.GetOwnerReferences() {
			switch ref.Kind {
			case "DaemonSet",
				"ReplicaSet",
				"ReplicationController",
				"StatefulSet":
				return false
			}
		}

	case "ReplicaSet.apps":
		for _, ref := range o.GetOwnerReferences() {
			switch ref.Kind {
			case "Deployment":
				return false
			}
		}

	case "ReplicationController":
		for _, ref := range o.GetOwnerReferences() {
			switch ref.Kind {
			case "DeploymentConfig":
				return false
			}
		}

	case "RoleBinding.rbac.authorization.k8s.io":
		// TODO: the intention here is to skip default rolebindings.
		matchRoleRef := func() bool {
			return reflect.DeepEqual(o.Object["roleRef"], map[string]interface{}{
				"apiGroup": "rbac.authorization.k8s.io",
				"kind":     "ClusterRole",
				"name":     strings.TrimSuffix(o.GetName(), "s"),
			})
		}

		switch o.GetName() {
		case "system:deployer", "system:deployers":
			if matchRoleRef() && reflect.DeepEqual(o.Object["subjects"], []interface{}{
				map[string]interface{}{
					"kind":      "ServiceAccount",
					"name":      "deployer",
					"namespace": ns,
				},
			}) {
				return false
			}
		case "system:image-builder", "system:image-builders":
			if matchRoleRef() && reflect.DeepEqual(o.Object["subjects"], []interface{}{
				map[string]interface{}{
					"kind":      "ServiceAccount",
					"name":      "builder",
					"namespace": ns,
				},
			}) {
				return false
			}
		case "system:image-puller", "system:image-pullers":
			if matchRoleRef() && reflect.DeepEqual(o.Object["subjects"], []interface{}{
				map[string]interface{}{
					"apiGroup": "rbac.authorization.k8s.io",
					"kind":     "Group",
					"name":     "system:serviceaccounts:" + ns,
				},
			}) {
				return false
			}
		}

	case "Secret":
		// TODO: the intention here is to skip automatically created secrets.
		switch jsonpath.MustCompile("$.type").MustGetString(o.Object) {
		case "kubernetes.io/dockercfg",
			"kubernetes.io/service-account-token":
			return !regexp.MustCompile("-[a-z0-9]{5}$").MatchString(o.GetName())
		}
		if _, found := o.GetAnnotations()["service.alpha.openshift.io/originating-service-name"]; found {
			return false
		}

	case "Service":
		if ns == "default" && o.GetName() == "kubernetes" {
			return false
		}

	case "ServiceAccount":
		// TODO: the intention here is to skip default service accounts.
		switch o.GetName() {
		case "builder",
			"default",
			"deployer":
			for _, field := range []string{"imagePullSecrets", "secrets"} {
				for _, secret := range jsonpath.MustCompile("$." + field + ".*.name").MustGetStrings(o.Object) {
					if !regexp.MustCompile("-[a-z0-9]{5}$").MatchString(secret) {
						return true
					}
				}
			}
			return false
		}

	case "Template.template.openshift.io":
		// TODO: openshift-ansible unwittingly puts service catalog templates in
		// other namespaces.  Need to file an issue upstream.  Workaround here.
		if ns != "openshift" {
			return false
		}
	}

	return true
}
