package admissioncontroller

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/origin/pkg/security/apis/security"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/kubernetes/pkg/apis/core"
)

func (ac *admissionController) handleSCC(w http.ResponseWriter, r *http.Request) {
	req, errcode := ac.getAdmissionReviewRequest(r)
	ac.log.Debug("New SCC validation request")
	if errcode != 0 {
		http.Error(w, http.StatusText(errcode), errcode)
		return
	}
	if req.Operation == admissionv1beta1.Delete {
		//allow Delete only on SCC which are not in the protected map
		_, protected := ac.bootstrapSCCs[req.Name]
		if protected {
			errs := []error{fmt.Errorf("Deleting of this SCC is not allowed")}
			ac.sendResult(errors.NewAggregate(errs), w, req.UID)
		} else {
			ac.sendResult(nil, w, req.UID)
		}
		return
	}
	//if Operation is Create,Update (Connect not configured in ValidatingWebhookConfiguration)
	scc, ok := req.Object.Object.(*security.SecurityContextConstraints)
	if !ok {
		ac.log.Errorf("SCC Object in request does not match declared GroupVersionKind")
		http.Error(w, "Bad request: SCC Object in request does not match declared GroupVersionKind", http.StatusBadRequest)
		return
	}
	sccTemplate, protected := ac.bootstrapSCCs[scc.Name]
	if protected {
		//SCC in the set of protected SCCs
		//only allow additional users and groups
		errs := ac.validateProtectedSCC(*scc, sccTemplate)
		ac.sendResult(errs, w, req.UID)
	} else {
		//SCC not in the set of protected SCCs
		//allow operation
		ac.sendResult(nil, w, req.UID)
	}
}

func (ac *admissionController) InitProtectedSCCs() map[string]security.SecurityContextConstraints {
	result := map[string]security.SecurityContextConstraints{
		"anyuid": {
			Priority:                 to.Int32Ptr(10),
			AllowPrivilegedContainer: false,
			DefaultAddCapabilities:   nil,
			RequiredDropCapabilities: []core.Capability{"MKNOD"},
			AllowedCapabilities:      nil,
			Volumes: []security.FSType{
				security.FSTypeConfigMap,
				security.FSTypeDownwardAPI,
				security.FSTypeEmptyDir,
				security.FSTypePersistentVolumeClaim,
				security.FSProjected,
				security.FSTypeSecret,
			},
			AllowHostNetwork:         false,
			AllowHostPorts:           false,
			AllowHostPID:             false,
			AllowHostIPC:             false,
			AllowPrivilegeEscalation: to.BoolPtr(true),
			TypeMeta:                 metav1.TypeMeta{},
			FSGroup: security.FSGroupStrategyOptions{
				Type: security.FSGroupStrategyRunAsAny,
			},
			Groups: []string{
				"system:cluster-admins",
			},

			RunAsUser: security.RunAsUserStrategyOptions{
				Type: security.RunAsUserStrategyRunAsAny,
			},
			SELinuxContext: security.SELinuxContextStrategyOptions{
				Type: security.SELinuxStrategyMustRunAs,
			},
			SupplementalGroups: security.SupplementalGroupsStrategyOptions{
				Type: security.SupplementalGroupsStrategyRunAsAny,
			},
		},
		"hostaccess": {
			Priority:                 nil,
			AllowPrivilegedContainer: false,
			DefaultAddCapabilities:   nil,
			RequiredDropCapabilities: []core.Capability{"KILL", "MKNOD", "SETUID", "SETGID"},
			AllowedCapabilities:      nil,
			Volumes: []security.FSType{
				security.FSTypeConfigMap,
				security.FSTypeDownwardAPI,
				security.FSTypeEmptyDir,
				security.FSTypeHostPath,
				security.FSTypePersistentVolumeClaim,
				security.FSProjected,
				security.FSTypeSecret,
			},
			AllowHostNetwork:         true,
			AllowHostPorts:           true,
			AllowHostPID:             true,
			AllowHostIPC:             true,
			AllowPrivilegeEscalation: to.BoolPtr(true),
			TypeMeta:                 metav1.TypeMeta{},
			FSGroup: security.FSGroupStrategyOptions{
				Type: security.FSGroupStrategyMustRunAs,
			},
			Groups: []string{},
			RunAsUser: security.RunAsUserStrategyOptions{
				Type: security.RunAsUserStrategyMustRunAsRange,
			},
			SELinuxContext: security.SELinuxContextStrategyOptions{
				Type: security.SELinuxStrategyMustRunAs,
			},
			SupplementalGroups: security.SupplementalGroupsStrategyOptions{
				Type: security.SupplementalGroupsStrategyRunAsAny,
			},
		},
		"hostmount-anyuid": {
			Priority:                 nil,
			AllowPrivilegedContainer: false,
			DefaultAddCapabilities:   nil,
			RequiredDropCapabilities: []core.Capability{"MKNOD"},
			AllowedCapabilities:      nil,
			Volumes: []security.FSType{
				security.FSTypeConfigMap,
				security.FSTypeDownwardAPI,
				security.FSTypeEmptyDir,
				security.FSTypeHostPath,
				security.FSTypeNFS,
				security.FSTypePersistentVolumeClaim,
				security.FSProjected,
				security.FSTypeSecret,
			},
			AllowHostNetwork:         false,
			AllowHostPorts:           false,
			AllowHostPID:             false,
			AllowHostIPC:             false,
			AllowPrivilegeEscalation: to.BoolPtr(true),
			TypeMeta:                 metav1.TypeMeta{},
			FSGroup: security.FSGroupStrategyOptions{
				Type: security.FSGroupStrategyRunAsAny,
			},
			Groups: []string{},
			RunAsUser: security.RunAsUserStrategyOptions{
				Type: security.RunAsUserStrategyRunAsAny,
			},
			SELinuxContext: security.SELinuxContextStrategyOptions{
				Type: security.SELinuxStrategyMustRunAs,
			},
			SupplementalGroups: security.SupplementalGroupsStrategyOptions{
				Type: security.SupplementalGroupsStrategyRunAsAny,
			},
			Users: []string{
				"system:serviceaccount:openshift-azure-monitoring:etcd-metrics",
				"system:serviceaccount:openshift-infra:pv-recycler-controller",
				"system:serviceaccount:kube-service-catalog:service-catalog-apiserver",
			},
		},
		"hostnetwork": {
			Priority:                 nil,
			AllowPrivilegedContainer: false,
			DefaultAddCapabilities:   nil,
			RequiredDropCapabilities: []core.Capability{"KILL", "MKNOD", "SETUID", "SETGID"},
			AllowedCapabilities:      nil,
			Volumes: []security.FSType{
				security.FSTypeConfigMap,
				security.FSTypeDownwardAPI,
				security.FSTypeEmptyDir,
				security.FSTypePersistentVolumeClaim,
				security.FSProjected,
				security.FSTypeSecret,
			},
			AllowHostNetwork:         true,
			AllowHostPorts:           true,
			AllowHostPID:             false,
			AllowHostIPC:             false,
			AllowPrivilegeEscalation: to.BoolPtr(true),
			TypeMeta:                 metav1.TypeMeta{},
			FSGroup: security.FSGroupStrategyOptions{
				Type: security.FSGroupStrategyMustRunAs,
			},
			Groups: []string{},
			RunAsUser: security.RunAsUserStrategyOptions{
				Type: security.RunAsUserStrategyMustRunAsRange,
			},
			SELinuxContext: security.SELinuxContextStrategyOptions{
				Type: security.SELinuxStrategyMustRunAs,
			},
			SupplementalGroups: security.SupplementalGroupsStrategyOptions{
				Type: security.SupplementalGroupsStrategyMustRunAs,
			},
		},
		"nonroot": {
			Priority:                 nil,
			AllowPrivilegedContainer: false,
			DefaultAddCapabilities:   nil,
			RequiredDropCapabilities: []core.Capability{"KILL", "MKNOD", "SETUID", "SETGID"},
			AllowedCapabilities:      nil,
			Volumes: []security.FSType{
				security.FSTypeConfigMap,
				security.FSTypeDownwardAPI,
				security.FSTypeEmptyDir,
				security.FSTypePersistentVolumeClaim,
				security.FSProjected,
				security.FSTypeSecret,
			},
			AllowHostNetwork:         false,
			AllowHostPorts:           false,
			AllowHostPID:             false,
			AllowHostIPC:             false,
			AllowPrivilegeEscalation: to.BoolPtr(true),
			TypeMeta:                 metav1.TypeMeta{},
			FSGroup: security.FSGroupStrategyOptions{
				Type: security.FSGroupStrategyRunAsAny,
			},
			Groups: []string{},
			RunAsUser: security.RunAsUserStrategyOptions{
				Type: security.RunAsUserStrategyMustRunAsNonRoot,
			},
			SELinuxContext: security.SELinuxContextStrategyOptions{
				Type: security.SELinuxStrategyMustRunAs,
			},
			SupplementalGroups: security.SupplementalGroupsStrategyOptions{
				Type: security.SupplementalGroupsStrategyRunAsAny,
			},
		},
		"privileged": {
			Priority:                 nil,
			AllowPrivilegedContainer: true,
			DefaultAddCapabilities:   nil,
			RequiredDropCapabilities: nil,
			AllowedCapabilities:      []core.Capability{"*"},
			Volumes: []security.FSType{
				security.FSTypeAll,
			},
			AllowHostNetwork:         true,
			AllowHostPorts:           true,
			AllowHostPID:             true,
			AllowHostIPC:             true,
			AllowPrivilegeEscalation: to.BoolPtr(true),
			TypeMeta:                 metav1.TypeMeta{},
			FSGroup: security.FSGroupStrategyOptions{
				Type: security.FSGroupStrategyRunAsAny,
			},
			Groups: []string{
				"system:cluster-admins",
				"system:nodes",
				"system:masters",
			},
			Users: []string{
				"system:admin",
				"system:serviceaccount:openshift-infra:build-controller",
				"system:serviceaccount:openshift-etcd:etcd-backup",
				"system:serviceaccount:openshift-azure-logging:log-analytics-agent",
				"system:serviceaccount:kube-system:sync",
			},
			RunAsUser: security.RunAsUserStrategyOptions{
				Type: security.RunAsUserStrategyRunAsAny,
			},
			SELinuxContext: security.SELinuxContextStrategyOptions{
				Type: security.SELinuxStrategyRunAsAny,
			},
			SupplementalGroups: security.SupplementalGroupsStrategyOptions{
				Type: security.SupplementalGroupsStrategyRunAsAny,
			},
			SeccompProfiles: []string{
				"*",
			},
			AllowedUnsafeSysctls: []string{
				"*",
			},
		},
		"restricted": {
			Priority:                 nil,
			AllowPrivilegedContainer: false,
			DefaultAddCapabilities:   nil,
			RequiredDropCapabilities: []core.Capability{"KILL", "MKNOD", "SETUID", "SETGID"},
			AllowedCapabilities:      nil,
			Volumes: []security.FSType{
				security.FSTypeConfigMap,
				security.FSTypeDownwardAPI,
				security.FSTypeEmptyDir,
				security.FSTypePersistentVolumeClaim,
				security.FSProjected,
				security.FSTypeSecret,
			},
			AllowHostNetwork:         false,
			AllowHostPorts:           false,
			AllowHostPID:             false,
			AllowHostIPC:             false,
			AllowPrivilegeEscalation: to.BoolPtr(true),
			TypeMeta:                 metav1.TypeMeta{},
			FSGroup: security.FSGroupStrategyOptions{
				Type: security.FSGroupStrategyMustRunAs,
			},
			Groups: []string{
				"system:authenticated",
			},
			Users: []string{},
			RunAsUser: security.RunAsUserStrategyOptions{
				Type: security.RunAsUserStrategyMustRunAsRange,
			},
			SELinuxContext: security.SELinuxContextStrategyOptions{
				Type: security.SELinuxStrategyMustRunAs,
			},
			SupplementalGroups: security.SupplementalGroupsStrategyOptions{
				Type: security.SupplementalGroupsStrategyRunAsAny,
			},
		},
	}
	return result
}

func contains(stringSlice []string, str string) bool {
	for _, e := range stringSlice {
		if str == e {
			return true
		}
	}
	return false
}

// validateProtectedSCC makes sure that nothing besides additional users or groups are
// different between the SCC and an SCCTemplate.
func (ac *admissionController) validateProtectedSCC(scc security.SecurityContextConstraints, sccTemplate security.SecurityContextConstraints) errors.Aggregate {
	var errs []error
	//Allow only if the new Groups are a superset of the template Groups
	for _, templateGroup := range sccTemplate.Groups {
		if !contains(scc.Groups, templateGroup) {
			errs = append(errs, fmt.Errorf("Removal of Group %s from SCC is not allowed", templateGroup))
			break
		}
	}
	//Allow only if the new Users are a superset of the template Groups
	for _, templateUser := range sccTemplate.Users {
		if !contains(scc.Users, templateUser) {
			errs = append(errs, fmt.Errorf("Removal of User %s from SCC is not allowed", templateUser))
			break
		}
	}
	localSccTemplate := sccTemplate.DeepCopy()
	//make sure the "owned-by-sync-pod" label is set
	localSccTemplate.Labels = scc.GetLabels()
	if localSccTemplate.Labels == nil {
		localSccTemplate.Labels = make(map[string]string)
	}
	localSccTemplate.Labels["azure.openshift.io/owned-by-sync-pod"] = "true"
	//ignore the remaining metadata in further comparison
	localSccTemplate.Name = scc.Name
	localSccTemplate.GenerateName = scc.GenerateName
	localSccTemplate.Namespace = scc.Namespace
	localSccTemplate.SelfLink = scc.SelfLink
	localSccTemplate.UID = scc.UID
	localSccTemplate.ResourceVersion = scc.ResourceVersion
	localSccTemplate.Generation = scc.Generation
	localSccTemplate.CreationTimestamp = scc.CreationTimestamp
	localSccTemplate.DeletionTimestamp = scc.DeletionTimestamp
	localSccTemplate.DeletionGracePeriodSeconds = scc.DeletionGracePeriodSeconds
	localSccTemplate.Annotations = scc.Annotations
	localSccTemplate.OwnerReferences = scc.OwnerReferences
	localSccTemplate.Initializers = scc.Initializers
	localSccTemplate.Finalizers = scc.Finalizers
	localSccTemplate.ClusterName = scc.ClusterName
	//ignore Users and Groups in further comparison
	localSccTemplate.Users = scc.Users
	localSccTemplate.Groups = scc.Groups

	if !reflect.DeepEqual(&scc, localSccTemplate) {
		errs = append(errs, fmt.Errorf("Modification of fields other than Users and Groups in the SCC is not allowed"))
	}
	return errors.NewAggregate(errs)
}
