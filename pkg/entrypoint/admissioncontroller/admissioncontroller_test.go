package admissioncontroller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"testing"

	_ "github.com/openshift/origin/pkg/api/install"
	"github.com/openshift/origin/pkg/security/apis/security"
	_ "github.com/openshift/origin/pkg/security/apis/security/install"
	"github.com/sirupsen/logrus"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/fake"
)

type fakeResponseWriter struct {
	statusCode int
	h          http.Header
	bytes.Buffer
}

func newFakeResponseWriter() *fakeResponseWriter {
	return &fakeResponseWriter{
		h:          map[string][]string{},
		statusCode: http.StatusOK,
	}
}

func (w *fakeResponseWriter) Header() http.Header {
	return w.h
}

func (w *fakeResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *fakeResponseWriter) Dump() {
	fmt.Printf("HTTP %d %s\r\n", w.statusCode, http.StatusText(w.statusCode))
	w.h.Write(os.Stdout)
	fmt.Print("\r\n")
	os.Stdout.Write(w.Bytes())
}

func TestHandleMalformedRequests(t *testing.T) {
	client := fake.NewSimpleClientset()

	restricted, err := getRestrictedSCC()
	if err != nil {
		t.Fatal(err)
	}

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	ac := &admissionController{
		client:     client,
		restricted: restricted,
		log:        logrus.NewEntry(logrus.StandardLogger()),
	}

	pod, err := json.Marshal(&corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "openshift",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Image: "regularimage",
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	req, err := json.Marshal(&admissionv1beta1.AdmissionReview{
		Request: &admissionv1beta1.AdmissionRequest{
			UID:      "uid",
			Kind:     metav1.GroupVersionKind{Version: "v1", Kind: "Pod"},
			Resource: metav1.GroupVersionResource{Version: "v1", Resource: "pods"},
			Object: runtime.RawExtension{
				Raw: pod,
			},
		}})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		changeInput func(*http.Request)
		response    *fakeResponseWriter
	}{
		{
			name: "bad request method",
			changeInput: func(input *http.Request) {
				input.Method = http.MethodGet
			},
			response: &fakeResponseWriter{
				statusCode: 405,
				h: http.Header{
					"X-Content-Type-Options": []string{"nosniff"},
					"Content-Type":           []string{"text/plain; charset=utf-8"},
				},
			},
		},
		{
			name: "bad Content-Type",
			changeInput: func(input *http.Request) {
				input.Header = http.Header{"Content-Type": []string{"application/pdf"}}
			},
			response: &fakeResponseWriter{
				statusCode: 415,
				h: http.Header{
					"X-Content-Type-Options": []string{"nosniff"},
					"Content-Type":           []string{"text/plain; charset=utf-8"},
				},
			},
		},
		{
			name: "bad content",
			changeInput: func(input *http.Request) {
				input.Body = ioutil.NopCloser(bytes.NewReader([]byte("this is not JSON")))
			},
			response: &fakeResponseWriter{
				statusCode: 400,
				h: http.Header{
					"X-Content-Type-Options": []string{"nosniff"},
					"Content-Type":           []string{"text/plain; charset=utf-8"},
				},
			},
		},
		{
			name: "no UID",
			changeInput: func(input *http.Request) {
				json, err := json.Marshal(&admissionv1beta1.AdmissionReview{
					Request: &admissionv1beta1.AdmissionRequest{
						UID:      "",
						Kind:     metav1.GroupVersionKind{Version: "v1", Kind: "Pod"},
						Resource: metav1.GroupVersionResource{Version: "v1", Resource: "pods"},
						Object: runtime.RawExtension{
							Raw: pod,
						},
					}})
				if err != nil {
					t.Fatal(err)
				}
				input.Body = ioutil.NopCloser(bytes.NewReader(json))
			},
			response: &fakeResponseWriter{
				statusCode: 400,
				h: http.Header{
					"X-Content-Type-Options": []string{"nosniff"},
					"Content-Type":           []string{"text/plain; charset=utf-8"},
				},
			},
		},
		{
			name: "no version, kind",
			changeInput: func(input *http.Request) {
				json, err := json.Marshal(&admissionv1beta1.AdmissionReview{
					Request: &admissionv1beta1.AdmissionRequest{
						UID:      "uid",
						Kind:     metav1.GroupVersionKind{},
						Resource: metav1.GroupVersionResource{Version: "v1", Resource: "pods"},
						Object: runtime.RawExtension{
							Raw: pod,
						},
					}})
				if err != nil {
					t.Fatal(err)
				}
				input.Body = ioutil.NopCloser(bytes.NewReader(json))
			},
			response: &fakeResponseWriter{
				statusCode: 400,
				h: http.Header{
					"X-Content-Type-Options": []string{"nosniff"},
					"Content-Type":           []string{"text/plain; charset=utf-8"},
				},
			},
		},
		{
			name: "wrong version, kind, good content",
			changeInput: func(input *http.Request) {
				json, err := json.Marshal(&admissionv1beta1.AdmissionReview{
					Request: &admissionv1beta1.AdmissionRequest{
						UID:      "uid",
						Kind:     metav1.GroupVersionKind{Group: "apps", Version: "v1", Kind: "DaemonSet"},
						Resource: metav1.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"},
						Object: runtime.RawExtension{
							Raw: []byte("{\"wrong\":true}"),
						},
					}})
				if err != nil {
					t.Fatal(err)
				}
				input.Body = ioutil.NopCloser(bytes.NewReader(json))
			},
			response: &fakeResponseWriter{
				statusCode: 200,
				h: http.Header{
					"Content-Type": []string{"application/json"},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := newFakeResponseWriter()
			request := &http.Request{
				Method: http.MethodPost,
				Header: http.Header{"Content-Type": []string{"application/json"}},
				Body:   ioutil.NopCloser(bytes.NewReader(req)),
			}
			test.changeInput(request)
			ac.handleWhitelist(w, request)
			if w.statusCode != test.response.statusCode {
				t.Errorf("handleWhitelist bad status code %d, expected %d", w.statusCode, test.response.statusCode)
			}
			if !reflect.DeepEqual(w.h, test.response.h) {
				t.Errorf("handleWhitelist got response headers %#v, expected %#v", w.h, test.response.h)
			}
		})
	}
}

func TestHandleWhitelistHappyPath(t *testing.T) {
	client := fake.NewSimpleClientset(&core.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
			Annotations: map[string]string{
				"openshift.io/sa.scc.uid-range": "1000/10",
				"openshift.io/sa.scc.mcs":       "mcs",
			},
		},
	})

	restricted, err := getRestrictedSCC()
	if err != nil {
		t.Fatal(err)
	}

	var whitelistedImages = []*regexp.Regexp{
		regexp.MustCompile("^whitelistedimage1$"),
		regexp.MustCompile("^whitelistedimage2$"),
	}
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	ac := &admissionController{
		client:            client,
		restricted:        restricted,
		whitelistedImages: whitelistedImages,
		log:               logrus.NewEntry(logrus.StandardLogger()),
	}

	for _, test := range []struct {
		name     string
		podSpec  corev1.PodSpec
		response *admissionv1beta1.AdmissionResponse
	}{
		{
			name: "regular non-privileged image, allow",
			podSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Image: "regularimage",
					},
				},
			},
			response: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: true,
				Result: &metav1.Status{
					Status: metav1.StatusSuccess,
				},
			},
		},
		{
			name: "regular privileged image, don't allow",
			podSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Image: "regularimage",
						SecurityContext: &corev1.SecurityContext{
							Privileged: &[]bool{true}[0],
						},
					},
				},
			},
			response: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: false,
				Result: &metav1.Status{
					Status:  metav1.StatusFailure,
					Message: "spec.containers[0].securityContext.privileged: Invalid value: true: Privileged containers are not allowed",
				},
			},
		},
		{
			name: "whitelisted non-privileged image, allow",
			podSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Image: "whitelistedimage1",
					},
				},
			},
			response: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: true,
				Result: &metav1.Status{
					Status: metav1.StatusSuccess,
				},
			},
		},
		{
			name: "whitelisted privileged image, allow",
			podSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Image: "whitelistedimage1",
						SecurityContext: &corev1.SecurityContext{
							Privileged: &[]bool{true}[0],
						},
					},
				},
			},
			response: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: true,
				Result: &metav1.Status{
					Status: metav1.StatusSuccess,
				},
			},
		},
		{
			name: "regular privileged image, annotated with master node selector, allow",
			podSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Image: "regulardimage",
						SecurityContext: &corev1.SecurityContext{
							Privileged: &[]bool{true}[0],
						},
					},
				},
				NodeSelector: map[string]string{
					"node-role.kubernetes.io/master": "true",
				},
			},
			response: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: true,
				Result: &metav1.Status{
					Status: metav1.StatusSuccess,
				},
			},
		},
		{
			name: "regular privileged image, annotated with infra node selector, allow",
			podSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Image: "regulardimage",
						SecurityContext: &corev1.SecurityContext{
							Privileged: &[]bool{true}[0],
						},
					},
				},
				NodeSelector: map[string]string{
					"node-role.kubernetes.io/infra": "true",
				},
			},
			response: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: true,
				Result: &metav1.Status{
					Status: metav1.StatusSuccess,
				},
			},
		},

		{
			name: "regular privileged image, assigned to master node, allow",
			podSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Image: "regulardimage",
						SecurityContext: &corev1.SecurityContext{
							Privileged: &[]bool{true}[0],
						},
					},
				},
				NodeName: "master-000000",
			},
			response: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: true,
				Result: &metav1.Status{
					Status: metav1.StatusSuccess,
				},
			},
		},
		{
			name: "regular privileged image, assigned to infra node, allow",
			podSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Image: "regulardimage",
						SecurityContext: &corev1.SecurityContext{
							Privileged: &[]bool{true}[0],
						},
					},
				},
				NodeName: "infra-123456-000002",
			},
			response: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: true,
				Result: &metav1.Status{
					Status: metav1.StatusSuccess,
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			pod, err := json.Marshal(&corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test",
				},
				Spec: test.podSpec,
			})
			if err != nil {
				t.Fatal(err)
			}

			req, err := json.Marshal(&admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					UID:      "uid",
					Kind:     metav1.GroupVersionKind{Version: "v1", Kind: "Pod"},
					Resource: metav1.GroupVersionResource{Version: "v1", Resource: "pods"},
					Object: runtime.RawExtension{
						Raw: pod,
					},
				}})
			if err != nil {
				t.Fatal(err)
			}

			r := &http.Request{
				Method: http.MethodPost,
				Header: http.Header{"Content-Type": []string{"application/json"}},
				Body:   ioutil.NopCloser(bytes.NewReader(req)),
			}

			w := newFakeResponseWriter()

			ac.handleWhitelist(w, r)

			if w.statusCode != 200 {
				t.Errorf("got status code %d, %s", w.statusCode, w.Buffer.String())
			}
			if !reflect.DeepEqual(w.Header(), http.Header{"Content-Type": []string{"application/json"}}) {
				t.Errorf("got header %#v", w.Header())
			}

			var rev *admissionv1beta1.AdmissionReview
			err = json.NewDecoder(w).Decode(&rev)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(rev.Response, test.response) {
				t.Errorf("got respose %#v", rev.Response)
			}
		})
	}
}

func TestHandleSCCHappyPathUgly(t *testing.T) {
	client := fake.NewSimpleClientset()

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	ac := &admissionController{
		client: client,
		log:    logrus.NewEntry(logrus.StandardLogger()),
	}
	ac.bootstrapSCCs = ac.InitProtectedSCCs()

	for _, test := range []struct {
		name     string
		scc      string
		response *admissionv1beta1.AdmissionResponse
	}{
		{
			name: "protected SCC, added user, allow",
			scc: `{
				"metadata": {
					"name": "hostmount-anyuid",
					"selfLink": "/apis/security.openshift.io/v1/securitycontextconstraints/hostmount-anyuid",
					"uid": "a615e699-fef2-11e9-b5af-000d3aaa0ca7",
					"resourceVersion": "3754",
					"creationTimestamp": "2019-11-04T11:02:53Z",
					"labels": {
						"azure.openshift.io/owned-by-sync-pod": "true"
					},
					"annotations": {
						"kubernetes.io/description": "hostmount-anyuid provides all the features of the restricted SCC but allows host mounts and any UID by a pod.  This is primarily used by the persistent volume recycler. WARNING: this SCC allows host file system access as any UID, including UID 0.  Grant with caution.",
						"openshift.io/reconcile-protect": "true"
					}
				},
				"priority": null,
				"allowPrivilegedContainer": false,
				"defaultAddCapabilities": null,
				"requiredDropCapabilities": [
					"MKNOD"
				],
				"allowedCapabilities": null,
				"allowHostDirVolumePlugin": true,
				"volumes": [
					"configMap",
					"downwardAPI",
					"emptyDir",
					"hostPath",
					"nfs",
					"persistentVolumeClaim",
					"projected",
					"secret"
				],
				"allowHostNetwork": false,
				"allowHostPorts": false,
				"allowHostPID": false,
				"allowHostIPC": false,
				"allowPrivilegeEscalation": true,
				"seLinuxContext": {
					"type": "MustRunAs"
				},
				"runAsUser": {
					"type": "RunAsAny"
				},
				"supplementalGroups": {
					"type": "RunAsAny"
				},
				"fsGroup": {
					"type": "RunAsAny"
				},
				"readOnlyRootFilesystem": false,
				"users": [
					"system:serviceaccount:openshift-azure-monitoring:etcd-metrics",
					"system:serviceaccount:openshift-infra:pv-recycler-controller",
					"system:serviceaccount:kube-service-catalog:service-catalog-apiserver",
					"myuser"
				],
				"groups": []
			}
			`,
			response: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: true,
				Result: &metav1.Status{
					Status: metav1.StatusSuccess,
				},
			},
		},
		{
			name: "protected SCC, remove system user, forbid",
			scc: `{
				"metadata": {
					"name": "hostmount-anyuid",
					"selfLink": "/apis/security.openshift.io/v1/securitycontextconstraints/hostmount-anyuid",
					"uid": "a615e699-fef2-11e9-b5af-000d3aaa0ca7",
					"resourceVersion": "3754",
					"creationTimestamp": "2019-11-04T11:02:53Z",
					"labels": {
						"azure.openshift.io/owned-by-sync-pod": "true"
					},
					"annotations": {
						"kubernetes.io/description": "hostmount-anyuid provides all the features of the restricted SCC but allows host mounts and any UID by a pod.  This is primarily used by the persistent volume recycler. WARNING: this SCC allows host file system access as any UID, including UID 0.  Grant with caution.",
						"openshift.io/reconcile-protect": "true"
					}
				},
				"priority": null,
				"allowPrivilegedContainer": false,
				"defaultAddCapabilities": null,
				"requiredDropCapabilities": [
					"MKNOD"
				],
				"allowedCapabilities": null,
				"allowHostDirVolumePlugin": true,
				"volumes": [
					"configMap",
					"downwardAPI",
					"emptyDir",
					"hostPath",
					"nfs",
					"persistentVolumeClaim",
					"projected",
					"secret"
				],
				"allowHostNetwork": false,
				"allowHostPorts": false,
				"allowHostPID": false,
				"allowHostIPC": false,
				"allowPrivilegeEscalation": true,
				"seLinuxContext": {
					"type": "MustRunAs"
				},
				"runAsUser": {
					"type": "RunAsAny"
				},
				"supplementalGroups": {
					"type": "RunAsAny"
				},
				"fsGroup": {
					"type": "RunAsAny"
				},
				"readOnlyRootFilesystem": false,
				"users": [
					"system:serviceaccount:openshift-azure-monitoring:etcd-metrics",
					"system:serviceaccount:openshift-infra:pv-recycler-controller"
				],
				"groups": []
			}
			`,
			response: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: false,
				Result: &metav1.Status{
					Status:  metav1.StatusFailure,
					Message: "Removal of User system:serviceaccount:kube-service-catalog:service-catalog-apiserver from SCC is not allowed",
				},
			},
		},
		{
			name: "protected SCC, added group, allow",
			scc: `{
				"allowHostIPC": false,
				"allowHostNetwork": false,
				"allowHostPID": false,
				"allowHostPorts": false,
				"allowPrivilegeEscalation": true,
				"allowPrivilegedContainer": false,
				"allowedCapabilities": null,
				"allowedFlexVolumes": null,
				"allowedUnsafeSysctls": null,
				"defaultAddCapabilities": null,
				"defaultAllowPrivilegeEscalation": null,
				"fSGroup": {
					"ranges": null,
					"type": "RunAsAny"
				},
				"forbiddenSysctls": null,
				"groups": [
					"system:cluster-admins",
					"myowngroup"
				],
				"metadata": {
					"creationTimestamp": null,
					"name": "anyuid",
					"labels": {
						"azure.openshift.io/owned-by-sync-pod": "true"
					}
				},
				"priority": 10,
				"readOnlyRootFilesystem": false,
				"requiredDropCapabilities": [
					"MKNOD"
				],
				"runAsUser": {
					"type": "RunAsAny",
					"uID": null,
					"uIDRangeMax": null,
					"uIDRangeMin": null
				},
				"seLinuxContext": {
					"seLinuxOptions": null,
					"type": "MustRunAs"
				},
				"seccompProfiles": null,
				"supplementalGroups": {
					"ranges": null,
					"type": "RunAsAny"
				},
				"typeMeta": {
					"apiVersion": "security.openshift.io/v1",
					"kind": "SecurityContextConstraints"
				},
				"users": null,
				"volumes": [
					"configMap",
					"downwardAPI",
					"emptyDir",
					"persistentVolumeClaim",
					"projected",
					"secret"
				]
			}
			`,
			response: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: true,
				Result: &metav1.Status{
					Status: metav1.StatusSuccess,
				},
			},
		},
		{
			name: "protected SCC, remove system group, forbid",
			scc: `{
				"allowHostIPC": false,
				"allowHostNetwork": false,
				"allowHostPID": false,
				"allowHostPorts": false,
				"allowPrivilegeEscalation": true,
				"allowPrivilegedContainer": false,
				"allowedCapabilities": null,
				"allowedFlexVolumes": null,
				"allowedUnsafeSysctls": null,
				"defaultAddCapabilities": null,
				"defaultAllowPrivilegeEscalation": null,
				"fSGroup": {
					"ranges": null,
					"type": "RunAsAny"
				},
				"forbiddenSysctls": null,
				"groups": null,
				"metadata": {
					"creationTimestamp": null,
					"name": "anyuid",
					"labels": {
						"azure.openshift.io/owned-by-sync-pod": "true"
					}
				},
				"priority": 10,
				"readOnlyRootFilesystem": false,
				"requiredDropCapabilities": [
					"MKNOD"
				],
				"runAsUser": {
					"type": "RunAsAny",
					"uID": null,
					"uIDRangeMax": null,
					"uIDRangeMin": null
				},
				"seLinuxContext": {
					"seLinuxOptions": null,
					"type": "MustRunAs"
				},
				"seccompProfiles": null,
				"supplementalGroups": {
					"ranges": null,
					"type": "RunAsAny"
				},
				"typeMeta": {
					"apiVersion": "security.openshift.io/v1",
					"kind": "SecurityContextConstraints"
				},
				"users": null,
				"volumes": [
					"configMap",
					"downwardAPI",
					"emptyDir",
					"persistentVolumeClaim",
					"projected",
					"secret"
				]
			}
			`,
			response: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: false,
				Result: &metav1.Status{
					Status:  metav1.StatusFailure,
					Message: "Removal of Group system:cluster-admins from SCC is not allowed",
				},
			},
		},
		{
			name: "protected SCC, changed allowprivilegedcontainer, forbid",
			scc: `{
				"allowHostIPC": false,
				"allowHostNetwork": false,
				"allowHostPID": false,
				"allowHostPorts": false,
				"allowPrivilegeEscalation": true,
				"allowPrivilegedContainer": true,
				"allowedCapabilities": null,
				"allowedFlexVolumes": null,
				"allowedUnsafeSysctls": null,
				"defaultAddCapabilities": null,
				"defaultAllowPrivilegeEscalation": null,
				"fSGroup": {
					"ranges": null,
					"type": "RunAsAny"
				},
				"forbiddenSysctls": null,
				"groups": [
					"system:cluster-admins"
				],
				"metadata": {
					"creationTimestamp": null,
					"name": "anyuid",
					"labels": {
						"azure.openshift.io/owned-by-sync-pod": "true"
					}
				},
				"priority": 10,
				"readOnlyRootFilesystem": false,
				"requiredDropCapabilities": [
					"MKNOD"
				],
				"runAsUser": {
					"type": "RunAsAny",
					"uID": null,
					"uIDRangeMax": null,
					"uIDRangeMin": null
				},
				"seLinuxContext": {
					"seLinuxOptions": null,
					"type": "MustRunAs"
				},
				"seccompProfiles": null,
				"supplementalGroups": {
					"ranges": null,
					"type": "RunAsAny"
				},
				"typeMeta": {
					"apiVersion": "security.openshift.io/v1",
					"kind": "SecurityContextConstraints"
				},
				"users": null,
				"volumes": [
					"configMap",
					"downwardAPI",
					"emptyDir",
					"persistentVolumeClaim",
					"projected",
					"secret"
				]
			}
			`,
			response: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: false,
				Result: &metav1.Status{
					Status:  metav1.StatusFailure,
					Message: "Modification of fields other than Users and Groups in the SCC is not allowed",
				},
			},
		},
		{
			name: "protected SCC, removed sync label, forbid",
			scc: `{
				"allowHostIPC": false,
				"allowHostNetwork": false,
				"allowHostPID": false,
				"allowHostPorts": false,
				"allowPrivilegeEscalation": true,
				"allowPrivilegedContainer": false,
				"allowedCapabilities": null,
				"allowedFlexVolumes": null,
				"allowedUnsafeSysctls": null,
				"defaultAddCapabilities": null,
				"defaultAllowPrivilegeEscalation": null,
				"fSGroup": {
					"ranges": null,
					"type": "RunAsAny"
				},
				"forbiddenSysctls": null,
				"groups": [
					"system:cluster-admins"
				],
				"metadata": {
					"creationTimestamp": null,
					"name": "anyuid"
				},
				"priority": 10,
				"readOnlyRootFilesystem": false,
				"requiredDropCapabilities": [
					"MKNOD"
				],
				"runAsUser": {
					"type": "RunAsAny",
					"uID": null,
					"uIDRangeMax": null,
					"uIDRangeMin": null
				},
				"seLinuxContext": {
					"seLinuxOptions": null,
					"type": "MustRunAs"
				},
				"seccompProfiles": null,
				"supplementalGroups": {
					"ranges": null,
					"type": "RunAsAny"
				},
				"typeMeta": {
					"apiVersion": "security.openshift.io/v1",
					"kind": "SecurityContextConstraints"
				},
				"users": null,
				"volumes": [
					"configMap",
					"downwardAPI",
					"emptyDir",
					"persistentVolumeClaim",
					"projected",
					"secret"
				]
			}
			`,
			response: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: false,
				Result: &metav1.Status{
					Status:  metav1.StatusFailure,
					Message: "Modification of fields other than Users and Groups in the SCC is not allowed",
				},
			},
		},
		{
			name: "unprotected SCC, allow",
			scc: `{
				"allowHostIPC": false,
				"allowHostNetwork": false,
				"allowHostPID": false,
				"allowHostPorts": false,
				"allowPrivilegeEscalation": true,
				"allowPrivilegedContainer": true,
				"allowedCapabilities": null,
				"allowedFlexVolumes": null,
				"allowedUnsafeSysctls": null,
				"defaultAddCapabilities": null,
				"defaultAllowPrivilegeEscalation": null,
				"fSGroup": {
					"ranges": null,
					"type": "RunAsAny"
				},
				"forbiddenSysctls": null,
				"groups": [
					"system:cluster-admins"
				],
				"metadata": {
					"creationTimestamp": null,
					"name": "notprotected"
				},
				"priority": 10,
				"readOnlyRootFilesystem": false,
				"requiredDropCapabilities": [
					"MKNOD"
				],
				"runAsUser": {
					"type": "RunAsAny",
					"uID": null,
					"uIDRangeMax": null,
					"uIDRangeMin": null
				},
				"seLinuxContext": {
					"seLinuxOptions": null,
					"type": "MustRunAs"
				},
				"seccompProfiles": null,
				"supplementalGroups": {
					"ranges": null,
					"type": "RunAsAny"
				},
				"typeMeta": {
					"apiVersion": "security.openshift.io/v1",
					"kind": "SecurityContextConstraints"
				},
				"users": null,
				"volumes": [
					"configMap",
					"downwardAPI",
					"emptyDir",
					"persistentVolumeClaim",
					"projected",
					"secret"
				]
			}
			`,
			response: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: true,
				Result: &metav1.Status{
					Status: metav1.StatusSuccess,
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			req, err := json.Marshal(&admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					UID:       "uid",
					Operation: admissionv1beta1.Update,
					Kind:      metav1.GroupVersionKind{Group: "security.openshift.io", Version: "v1", Kind: "SecurityContextConstraints"},
					Resource:  metav1.GroupVersionResource{Group: "security.openshift.io", Version: "v1", Resource: "securitycontextconstraints"},
					Object: runtime.RawExtension{
						Raw: []byte(test.scc),
					},
				}})
			if err != nil {
				t.Fatal(err)
			}

			r := &http.Request{
				Method: http.MethodPost,
				Header: http.Header{"Content-Type": []string{"application/json"}},
				Body:   ioutil.NopCloser(bytes.NewReader(req)),
			}

			w := newFakeResponseWriter()

			ac.handleSCC(w, r)

			if w.statusCode != 200 {
				t.Errorf("got status code %d, %s", w.statusCode, w.Buffer.String())
			}
			if !reflect.DeepEqual(w.Header(), http.Header{"Content-Type": []string{"application/json"}}) {
				t.Errorf("got header %#v", w.Header())
			}

			var rev *admissionv1beta1.AdmissionReview
			err = json.NewDecoder(w).Decode(&rev)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(rev.Response, test.response) {
				t.Errorf("got respose %#v, expected %#v", rev.Response, test.response)
				t.Errorf("status %#v, expected %#v", rev.Response.Result, test.response.Result)
			}
		})
	}
}

func TestHandleSCCHappyPath(t *testing.T) {
	baseScc := []byte(`{
		"metadata": {
			"name": "hostmount-anyuid",
			"selfLink": "/apis/security.openshift.io/v1/securitycontextconstraints/hostmount-anyuid",
			"uid": "a615e699-fef2-11e9-b5af-000d3aaa0ca7",
			"resourceVersion": "3754",
			"creationTimestamp": "2019-11-04T11:02:53Z",
			"labels": {
				"azure.openshift.io/owned-by-sync-pod": "true"
			},
			"annotations": {
				"kubernetes.io/description": "hostmount-anyuid provides all the features of the restricted SCC but allows host mounts and any UID by a pod.  This is primarily used by the persistent volume recycler. WARNING: this SCC allows host file system access as any UID, including UID 0.  Grant with caution.",
				"openshift.io/reconcile-protect": "true"
			}
		},
		"priority": null,
		"allowPrivilegedContainer": false,
		"defaultAddCapabilities": null,
		"requiredDropCapabilities": [
			"MKNOD"
		],
		"allowedCapabilities": null,
		"allowHostDirVolumePlugin": true,
		"volumes": [
			"configMap",
			"downwardAPI",
			"emptyDir",
			"hostPath",
			"nfs",
			"persistentVolumeClaim",
			"projected",
			"secret"
		],
		"allowHostNetwork": false,
		"allowHostPorts": false,
		"allowHostPID": false,
		"allowHostIPC": false,
		"allowPrivilegeEscalation": true,
		"seLinuxContext": {
			"type": "MustRunAs"
		},
		"runAsUser": {
			"type": "RunAsAny"
		},
		"supplementalGroups": {
			"type": "RunAsAny"
		},
		"fsGroup": {
			"type": "RunAsAny"
		},
		"readOnlyRootFilesystem": false,
		"users": [
			"system:serviceaccount:kube-service-catalog:service-catalog-apiserver",
			"system:serviceaccount:openshift-azure-monitoring:etcd-metrics",
			"system:serviceaccount:openshift-infra:pv-recycler-controller"
		],
		"groups": []
	}
	`)

	gvk := schema.GroupVersionKind{Group: "security.openshift.io", Version: "v1", Kind: "SecurityContextConstraints"}

	tests := []struct {
		name           string
		changeInput    func(input *security.SecurityContextConstraints) *security.SecurityContextConstraints
		expectedResult *admissionv1beta1.AdmissionResponse
	}{
		{
			name: "protected SCC, add user and group, allow",
			changeInput: func(input *security.SecurityContextConstraints) *security.SecurityContextConstraints {
				input.Users = append(input.Users, "testuser")
				input.Groups = append(input.Users, "testgroup")
				return input
			},
			expectedResult: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: true,
				Result: &metav1.Status{
					Status:  metav1.StatusSuccess,
					Message: "",
				},
			},
		},
		{
			name: "protected SCC, removed system users, forbid",
			changeInput: func(input *security.SecurityContextConstraints) *security.SecurityContextConstraints {
				input.Users = []string{
					"system:serviceaccount:kube-service-catalog:service-catalog-apiserver",
					"system:serviceaccount:openshift-azure-monitoring:etcd-metrics",
				}
				return input
			},
			expectedResult: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: false,
				Result: &metav1.Status{
					Status:  metav1.StatusFailure,
					Message: "Removal of User system:serviceaccount:openshift-infra:pv-recycler-controller from SCC is not allowed",
				},
			},
		},
		{
			name: "protected SCC, changed allowprivilegedcontaine, forbid",
			changeInput: func(input *security.SecurityContextConstraints) *security.SecurityContextConstraints {
				input.AllowPrivilegedContainer = true
				return input
			},
			expectedResult: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: false,
				Result: &metav1.Status{
					Status:  metav1.StatusFailure,
					Message: "Modification of fields other than Users and Groups in the SCC is not allowed",
				},
			},
		},
		{
			name: "protected SCC, removed sync label, forbid",
			changeInput: func(input *security.SecurityContextConstraints) *security.SecurityContextConstraints {
				input.ObjectMeta.Labels = nil
				return input
			},
			expectedResult: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: true,
				Result: &metav1.Status{
					Status:  metav1.StatusSuccess,
					Message: "",
				},
			},
		},
		{
			name: "unprotected SCC, allow",
			changeInput: func(input *security.SecurityContextConstraints) *security.SecurityContextConstraints {
				input.ObjectMeta.Name = "notprotected"
				return input
			},
			expectedResult: &admissionv1beta1.AdmissionResponse{
				UID:     "uid",
				Allowed: true,
				Result: &metav1.Status{
					Status:  metav1.StatusSuccess,
					Message: "",
				},
			},
		},
	}

	client := fake.NewSimpleClientset()

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	ac := &admissionController{
		client: client,
		log:    logrus.NewEntry(logrus.StandardLogger()),
	}
	ac.bootstrapSCCs = ac.InitProtectedSCCs()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			o, _, err := codec.Decode(baseScc, &gvk, nil)
			if err != nil {
				t.Fatal(err)
			}

			scc := test.changeInput(o.(*security.SecurityContextConstraints))

			scc.TypeMeta.APIVersion = "security.openshift.io/v1"
			scc.TypeMeta.Kind = "SecurityContextConstraints"
			buf := new(bytes.Buffer)
			err = codec.Encode(scc, buf)
			if err != nil {
				t.Fatal(err)
			}
			//TODO find an encoder which puts metav1.ObjectMeta
			//into the "metadata" json field
			//instead of just composing its fields into the SCC

			req, err := json.Marshal(&admissionv1beta1.AdmissionReview{
				Request: &admissionv1beta1.AdmissionRequest{
					UID:       "uid",
					Operation: admissionv1beta1.Update,
					Kind:      metav1.GroupVersionKind{Group: "security.openshift.io", Version: "v1", Kind: "SecurityContextConstraints"},
					Resource:  metav1.GroupVersionResource{Group: "security.openshift.io", Version: "v1", Resource: "securitycontextconstraints"},
					Object: runtime.RawExtension{
						Raw: buf.Bytes(),
					},
				}})
			if err != nil {
				t.Fatal(err)
			}
			r := &http.Request{
				Method: http.MethodPost,
				Header: http.Header{"Content-Type": []string{"application/json"}},
				Body:   ioutil.NopCloser(bytes.NewReader(req)),
			}
			w := newFakeResponseWriter()

			//function under test
			ac.handleSCC(w, r)

			if w.statusCode != 200 {
				t.Errorf("got status code %d, %s", w.statusCode, w.Buffer.String())
			}
			if !reflect.DeepEqual(w.Header(), http.Header{"Content-Type": []string{"application/json"}}) {
				t.Errorf("got header %#v", w.Header())
			}

			var rev *admissionv1beta1.AdmissionReview
			err = json.NewDecoder(w).Decode(&rev)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(rev.Response, test.expectedResult) {
				t.Errorf("got respose %#v, expected %#v", rev.Response, test.expectedResult)
				t.Errorf("status %#v, expected %#v", rev.Response.Result, test.expectedResult.Result)
			}
		})
	}
}
