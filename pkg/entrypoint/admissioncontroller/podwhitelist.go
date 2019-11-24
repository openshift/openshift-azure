package admissioncontroller

import (
	"fmt"
	"net/http"
	"strings"

	oapps "github.com/openshift/origin/pkg/apps/apis/apps"
	"github.com/openshift/origin/pkg/security/apiserver/securitycontextconstraints"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/kubernetes/pkg/apis/admission"
	"k8s.io/kubernetes/pkg/apis/apps"
	"k8s.io/kubernetes/pkg/apis/batch"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/extensions"
)

func (ac *admissionController) handleWhitelist(w http.ResponseWriter, r *http.Request) {
	req, errcode := ac.getAdmissionRequest(r)
	if errcode != 0 {
		http.Error(w, http.StatusText(errcode), errcode)
		return
	}

	errs, err := ac.validateWhitelistRequest(req)
	if err != nil {
		ac.log.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ac.sendReview(w, req, errs.ToAggregate())
}

func (ac *admissionController) validateWhitelistRequest(req *admission.AdmissionRequest) (field.ErrorList, error) {
	if strings.HasPrefix(req.Namespace, "openshift-") ||
		strings.HasPrefix(req.Namespace, "kube-") ||
		req.Namespace == "default" {
		return nil, nil
	}

	if req.UserInfo.Username == "system:serviceaccount:openshift-infra:build-controller" {
		return nil, nil
	}

	path, pod, err := ac.unpack(req.Object)
	if err != nil {
		return nil, err
	}

	return ac.validateWhitelistPod(path, pod)
}

func (ac *admissionController) validateWhitelistPod(path *field.Path, pod *core.Pod) (field.ErrorList, error) {
	// fast path
	isValid, err := ac.podIsValidUnderRestrictedSCC(pod)
	if err != nil || isValid {
		return nil, err
	}

	errs, err := ac.validateWhitelistContainers(child(path, "spec", "containers"), pod, pod.Spec.Containers)
	if err != nil {
		return nil, err
	}

	validationErrs, err := ac.validateWhitelistContainers(child(path, "spec", "initContainers"), pod, pod.Spec.InitContainers)
	if err != nil {
		return nil, err
	}

	return append(errs, validationErrs...), nil
}

func (ac *admissionController) podIsValidUnderRestrictedSCC(pod *core.Pod) (bool, error) {
	provider, _, err := securitycontextconstraints.CreateProviderFromConstraint(pod.Namespace, nil, ac.sccs["restricted"], ac.client)
	if err != nil {
		return false, err
	}

	errs, err := securitycontextconstraints.AssignSecurityContext(provider, pod, field.NewPath(fmt.Sprintf("provider %s: ", provider.GetSCCName()))), nil
	return errs == nil, err
}

func (ac *admissionController) validateWhitelistContainers(path *field.Path, pod *core.Pod, cs []core.Container) (errs field.ErrorList, err error) {
	for i, c := range cs {
		validationErr, err := ac.validateWhitelistContainer(path.Index(i), pod, &c)
		if err != nil {
			return nil, err
		}

		if validationErr != nil {
			errs = append(errs, validationErr)
		}
	}

	return
}

func (ac *admissionController) validateWhitelistContainer(path *field.Path, originalPod *core.Pod, c *core.Container) (*field.Error, error) {
	pod := originalPod.DeepCopy()
	pod.Spec.Containers, pod.Spec.InitContainers = []core.Container{*c}, nil

	isValid, err := ac.podIsValidUnderRestrictedSCC(pod)
	if err != nil || isValid {
		return nil, err
	}

	if ac.imageIsWhitelisted(c.Image) {
		return nil, nil
	}

	return field.Forbidden(path, "requires privileges but image is not whitelisted on platform"), nil
}

func (ac *admissionController) imageIsWhitelisted(image string) bool {
	for _, rx := range ac.imageWhitelist {
		if rx.MatchString(image) {
			return true
		}
	}

	return false
}

func (ac *admissionController) unpack(o runtime.Object) (path *field.Path, pod *core.Pod, err error) {
	path = field.NewPath("spec", "template")

	switch o := o.(type) {
	case *apps.StatefulSet:
		pod = &core.Pod{ObjectMeta: o.Spec.Template.ObjectMeta, Spec: o.Spec.Template.Spec}
		pod.Namespace = o.Namespace
	case *batch.CronJob:
		pod = &core.Pod{ObjectMeta: o.Spec.JobTemplate.Spec.Template.ObjectMeta, Spec: o.Spec.JobTemplate.Spec.Template.Spec}
		pod.Namespace = o.Namespace
		path = field.NewPath("spec", "jobTemplate", "spec", "template")
	case *batch.Job:
		pod = &core.Pod{ObjectMeta: o.Spec.Template.ObjectMeta, Spec: o.Spec.Template.Spec}
		pod.Namespace = o.Namespace
	case *core.Pod:
		pod = &core.Pod{ObjectMeta: o.ObjectMeta, Spec: o.Spec}
		pod.Namespace = o.Namespace
		path = nil
	case *core.ReplicationController:
		pod = &core.Pod{ObjectMeta: o.Spec.Template.ObjectMeta, Spec: o.Spec.Template.Spec}
		pod.Namespace = o.Namespace
	case *extensions.DaemonSet:
		pod = &core.Pod{ObjectMeta: o.Spec.Template.ObjectMeta, Spec: o.Spec.Template.Spec}
		pod.Namespace = o.Namespace
	case *extensions.Deployment:
		pod = &core.Pod{ObjectMeta: o.Spec.Template.ObjectMeta, Spec: o.Spec.Template.Spec}
		pod.Namespace = o.Namespace
	case *extensions.ReplicaSet:
		pod = &core.Pod{ObjectMeta: o.Spec.Template.ObjectMeta, Spec: o.Spec.Template.Spec}
		pod.Namespace = o.Namespace
	case *oapps.DeploymentConfig:
		pod = &core.Pod{ObjectMeta: o.Spec.Template.ObjectMeta, Spec: o.Spec.Template.Spec}
		pod.Namespace = o.Namespace
	default:
		return nil, nil, fmt.Errorf("unimplemented gvk %v", o.GetObjectKind().GroupVersionKind())
	}

	return
}

func child(p *field.Path, names ...string) *field.Path {
	if p == nil {
		return field.NewPath(names[0], names[1:]...)
	}

	return p.Child(names[0], names[1:]...)
}
