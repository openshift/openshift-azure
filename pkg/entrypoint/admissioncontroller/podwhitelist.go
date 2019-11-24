package admissioncontroller

import (
	"fmt"
	"net/http"
	"strings"

	_ "github.com/openshift/origin/pkg/api/install"
	oapps "github.com/openshift/origin/pkg/apps/apis/apps"
	"github.com/openshift/origin/pkg/security/apiserver/securitycontextconstraints"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/kubernetes/pkg/apis/apps"
	"k8s.io/kubernetes/pkg/apis/batch"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/extensions"
)

//unpack unpacks the PodSpec, ObjectMeta and Namespace, and wraps it in a Pod object
func unpack(o runtime.Object) (*core.Pod, error) {
	switch o := o.(type) {
	case *core.Pod:
		return o, nil
	case *extensions.DaemonSet:
		pod := &core.Pod{ObjectMeta: o.Spec.Template.ObjectMeta, Spec: o.Spec.Template.Spec}
		pod.Namespace = o.Namespace
		return pod, nil
	case *extensions.ReplicaSet:
		pod := &core.Pod{ObjectMeta: o.Spec.Template.ObjectMeta, Spec: o.Spec.Template.Spec}
		pod.Namespace = o.Namespace
		return pod, nil
	case *apps.StatefulSet:
		pod := &core.Pod{ObjectMeta: o.Spec.Template.ObjectMeta, Spec: o.Spec.Template.Spec}
		pod.Namespace = o.Namespace
		return pod, nil
	case *batch.Job:
		pod := &core.Pod{ObjectMeta: o.Spec.Template.ObjectMeta, Spec: o.Spec.Template.Spec}
		pod.Namespace = o.Namespace
		return pod, nil
	case *batch.CronJob:
		pod := &core.Pod{ObjectMeta: o.Spec.JobTemplate.Spec.Template.ObjectMeta, Spec: o.Spec.JobTemplate.Spec.Template.Spec}
		pod.Namespace = o.Namespace
		return pod, nil
	case *oapps.DeploymentConfig:
		pod := &core.Pod{ObjectMeta: o.Spec.Template.ObjectMeta, Spec: o.Spec.Template.Spec}
		pod.Namespace = o.Namespace
		return pod, nil
	case *extensions.Deployment:
		pod := &core.Pod{ObjectMeta: o.Spec.Template.ObjectMeta, Spec: o.Spec.Template.Spec}
		pod.Namespace = o.Namespace
		return pod, nil
	default:
		return nil, fmt.Errorf("unimplemented gvk %v", o.GetObjectKind().GroupVersionKind())
	}
}

func (ac *admissionController) handleWhitelist(w http.ResponseWriter, r *http.Request) {
	req, errcode := ac.getAdmissionReviewRequest(r)
	if errcode != 0 {
		http.Error(w, http.StatusText(errcode), errcode)
		return
	}
	if req.UID == "" || req.Kind.Version == "" || req.Kind.Kind == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	//whitelist system namespaces
	systemNamespaces := map[string]bool{
		"default":                           true,
		"kube-public":                       true,
		"kube-service-catalog":              true,
		"kube-system":                       true,
		"openshift-ansible-service-broker":  true,
		"openshift-azure":                   true,
		"openshift-azure-branding":          true,
		"openshift-azure-logging":           true,
		"openshift-azure-monitoring":        true,
		"openshift-console":                 true,
		"openshift-etcd":                    true,
		"openshift-infra":                   true,
		"openshift-logging":                 true,
		"openshift-monitoring":              true,
		"openshift-node":                    true,
		"openshift-sdn":                     true,
		"openshift-template-service-broker": true,
		"openshift-web-console":             true,
	}
	if _, present := systemNamespaces[req.Namespace]; present {
		ac.sendResult(nil, w, req.UID)
		return
	}
	//whitelist builds
	if req.UserInfo.Username == "system:serviceaccount:openshift-infra:build-controller" {
		ac.sendResult(nil, w, req.UID)
		return
	}
	pod, err := unpack(req.Object.Object)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
		return
	}
	ac.checkPodSpec(pod, w, req.UID)
}

//checkPodSpec checks if the Pod spec is either whitelisted or will match the restricted scc, then prepares an HTTP response
// interface{} is used to allow core.Pod from both the Openshift and Kubernetes APIs
func (ac *admissionController) checkPodSpec(pod *core.Pod, w http.ResponseWriter, uid types.UID) {
	errs, err := ac.validatePodAgainstSCC(pod, pod.Namespace)
	if err != nil {
		ac.log.Errorf("Validation error: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ac.log.Debugf("Review complete")
	ac.sendResult(errs.ToAggregate(), w, uid)
}

func (ac *admissionController) validatePodAgainstSCC(pod *core.Pod, namespace string) (field.ErrorList, error) {
	if ac.podSpecIsWhitelisted(&pod.Spec) {
		ac.log.Debugf("Pod is whitelisted")
		return nil, nil
	}
	ac.log.Debugf("Pod is not whitelisted")
	provider, _, err := securitycontextconstraints.CreateProviderFromConstraint(namespace, nil, ac.restricted, ac.client)
	if err != nil {
		return nil, err
	}

	return securitycontextconstraints.AssignSecurityContext(provider, pod, field.NewPath(fmt.Sprintf("provider %s: ", provider.GetSCCName()))), nil
}

// podIsWhitelisted returns true if all images of all containers are whitelisted
func (ac *admissionController) podSpecIsWhitelisted(spec *core.PodSpec) bool {
	if spec.NodeSelector != nil {
		ac.log.Debugf("NodeSelector not nil: %v", spec.NodeSelector)
		if spec.NodeSelector["node-role.kubernetes.io/master"] == "true" || spec.NodeSelector["node-role.kubernetes.io/infra"] == "true" {
			return true
		}
	}
	//nodeSelector is not sent in the static Pod review request, but the Node is available
	if strings.HasPrefix(spec.NodeName, "master-") || strings.HasPrefix(spec.NodeName, "infra-") {
		//if it's a pod assigned to a master or infra node it should be able to run
		return true
	}
	containers := append([]core.Container{}, spec.Containers...)
	containers = append(containers, spec.InitContainers...)
	for _, c := range containers {
		ac.log.Debugf("Image %s", c.Image)
		if !ac.imageIsWhitelisted(c.Image) {
			return false
		}
	}

	return true
}

func (ac *admissionController) imageIsWhitelisted(image string) bool {
	for _, rx := range ac.whitelistedImages {
		if rx.MatchString(image) {
			return true
		}
	}
	return false
}
