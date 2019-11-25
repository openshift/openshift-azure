package admissioncontroller

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"reflect"
	"regexp"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/admission"
	"k8s.io/kubernetes/pkg/apis/apps"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/extensions"
)

func TestWhitelist(t *testing.T) {
	l := newTestListener()
	defer l.Close()

	ac := &admissionController{
		l:      l,
		log:    logrus.NewEntry(logrus.StandardLogger()),
		cs:     cs,
		client: client,
		imageWhitelist: []*regexp.Regexp{
			regexp.MustCompile("^whitelisted$"),
		},
		sccs: sccs,
	}

	go ac.run()

	pool := x509.NewCertPool()
	pool.AddCert(cs.Config.Certificates.Ca.Cert)

	c := &http.Client{
		Transport: &http.Transport{
			Dial: l.Dial,
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{
					{
						Certificate: [][]byte{
							cs.Config.Certificates.AroAdmissionControllerClient.Cert.Raw,
						},
						PrivateKey: cs.Config.Certificates.AroAdmissionControllerClient.Key,
					},
				},
				RootCAs: pool,
			},
		},
	}

	restricted := func() *admission.AdmissionReview {
		return &admission.AdmissionReview{
			Request: &admission.AdmissionRequest{
				Namespace: "customer-namespace",
				Object:    &core.Pod{},
			},
		}
	}

	privileged := func() *admission.AdmissionReview {
		return &admission.AdmissionReview{
			Request: &admission.AdmissionRequest{
				Namespace: "customer-namespace",
				Object: &core.Pod{
					Spec: core.PodSpec{
						Containers: []core.Container{
							{
								SecurityContext: &core.SecurityContext{
									Privileged: to.BoolPtr(true),
								},
							},
							{
								Image: "whitelisted",
								SecurityContext: &core.SecurityContext{
									Privileged: to.BoolPtr(true),
								},
							},
						},
						InitContainers: []core.Container{
							{
								Image: "whitelisted",
								SecurityContext: &core.SecurityContext{
									Privileged: to.BoolPtr(true),
								},
							},
							{
								SecurityContext: &core.SecurityContext{
									Privileged: to.BoolPtr(true),
								},
							},
						},
					},
				},
			},
		}
	}

	privilegedWhitelisted := func() *admission.AdmissionReview {
		return &admission.AdmissionReview{
			Request: &admission.AdmissionRequest{
				Namespace: "customer-namespace",
				Object: &core.Pod{
					Spec: core.PodSpec{
						Containers: []core.Container{
							{
								Image: "whitelisted",
								SecurityContext: &core.SecurityContext{
									Privileged: to.BoolPtr(true),
								},
							},
						},
						InitContainers: []core.Container{
							{
								Image: "whitelisted",
								SecurityContext: &core.SecurityContext{
									Privileged: to.BoolPtr(true),
								},
							},
						},
					},
				},
			},
		}
	}

	for _, tt := range []struct {
		name        string
		request     func() *admission.AdmissionReview
		wantMessage string
	}{
		{
			name:    "restricted pod allowed",
			request: restricted,
		},
		{
			name: "restricted deployment allowed",
			request: func() *admission.AdmissionReview {
				review := restricted()
				review.Request.Object = &extensions.Deployment{
					Spec: extensions.DeploymentSpec{
						Template: core.PodTemplateSpec{
							Spec: review.Request.Object.(*core.Pod).Spec,
						},
					},
				}
				return review
			},
		},
		{
			name:        "non-whitelisted privileged pod not allowed",
			request:     privileged,
			wantMessage: "[spec.containers[0]: Forbidden: requires privileges but image is not whitelisted on platform, spec.initContainers[1]: Forbidden: requires privileges but image is not whitelisted on platform]",
		},
		{
			name: "non-whitelisted privileged stateful set not allowed",
			request: func() *admission.AdmissionReview {
				review := privileged()
				review.Request.Object = &apps.StatefulSet{
					Spec: apps.StatefulSetSpec{
						Template: core.PodTemplateSpec{
							Spec: review.Request.Object.(*core.Pod).Spec,
						},
					},
				}
				return review
			},
			wantMessage: "[spec.template.spec.containers[0]: Forbidden: requires privileges but image is not whitelisted on platform, spec.template.spec.initContainers[1]: Forbidden: requires privileges but image is not whitelisted on platform]",
		},
		{
			name:    "whitelisted privileged pod allowed",
			request: privilegedWhitelisted,
		},
		{
			name: "whitelisted privileged replica set allowed",
			request: func() *admission.AdmissionReview {
				review := privilegedWhitelisted()
				review.Request.Object = &extensions.ReplicaSet{
					Spec: extensions.ReplicaSetSpec{
						Template: core.PodTemplateSpec{
							Spec: review.Request.Object.(*core.Pod).Spec,
						},
					},
				}
				return review
			},
		},
		{
			name: "privileged pod in system namespace allowed",
			request: func() *admission.AdmissionReview {
				review := privileged()
				review.Request.Namespace = "kube-system"
				return review
			},
		},
		{
			name: "privileged pod from build controller allowed",
			request: func() *admission.AdmissionReview {
				review := privileged()
				review.Request.UserInfo.Username = "system:serviceaccount:openshift-infra:build-controller"
				return review
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request := tt.request()
			request.Request.UID = "uid"

			wantResponse := &admission.AdmissionReview{
				Response: &admission.AdmissionResponse{
					UID:     "uid",
					Allowed: true,
					Result: &metav1.Status{
						Status: metav1.StatusSuccess,
					},
				},
			}

			if tt.wantMessage != "" {
				wantResponse = &admission.AdmissionReview{
					Response: &admission.AdmissionResponse{
						UID: "uid",
						Result: &metav1.Status{
							Status:  metav1.StatusFailure,
							Message: tt.wantMessage,
						},
					},
				}
			}

			response, err := doRequest(c, "https://server/podwhitelist", request)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(response, wantResponse) {
				c := spew.ConfigState{
					DisableMethods: true,
				}
				t.Error(c.Sdump(response))
			}
		})
	}
}
