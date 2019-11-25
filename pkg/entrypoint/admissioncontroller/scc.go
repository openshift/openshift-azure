package admissioncontroller

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/openshift/origin/pkg/security/apis/security"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/kubernetes/pkg/apis/admission"
)

func (ac *admissionController) handleSCC(w http.ResponseWriter, r *http.Request) {
	req, errcode := ac.getAdmissionRequest(r)
	if errcode != 0 {
		http.Error(w, http.StatusText(errcode), errcode)
		return
	}

	errs, err := ac.validateSCCRequest(req)
	if err != nil {
		ac.log.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ac.sendReview(w, req, errs)
}

func (ac *admissionController) validateSCCRequest(req *admission.AdmissionRequest) (errors.Aggregate, error) {
	template, found := ac.sccs[req.Name]
	if !found { // not controlled by us, they can do what they like
		return nil, nil
	}

	if req.Operation == admission.Delete {
		return errors.NewAggregate([]error{fmt.Errorf("system SCC %q may not be deleted", req.Name)}), nil
	}

	scc, ok := req.Object.(*security.SecurityContextConstraints)
	if !ok {
		return nil, fmt.Errorf("received object of type %T", req.Object)
	}

	var errs []error
	for _, g := range template.Groups {
		if !contains(scc.Groups, g) {
			errs = append(errs, field.Required(field.NewPath("groups"), fmt.Sprintf("must include group %s", g)))
		}
	}
	scc.Groups = template.Groups

	for _, u := range template.Users {
		if !contains(scc.Users, u) {
			errs = append(errs, field.Required(field.NewPath("users"), fmt.Sprintf("must include user %s", u)))
		}
	}
	scc.Users = template.Users

	delete(scc.Labels, "openshift.io/reconcile-protect")

	scc.SelfLink = template.SelfLink
	scc.UID = template.UID
	scc.ResourceVersion = template.ResourceVersion
	scc.Generation = template.Generation
	scc.CreationTimestamp = template.CreationTimestamp

	// TODO: should do this field-wise
	if !reflect.DeepEqual(scc, template) {
		errs = append(errs, field.Invalid(field.NewPath(""), "", fmt.Sprint("may not modify fields other than users, groups and label labels.openshift.io/reconcile-protect")))
	}

	return errors.NewAggregate(errs), nil
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}

	return false
}
