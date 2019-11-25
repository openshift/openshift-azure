package admissioncontroller

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"reflect"
	"regexp"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/openshift/origin/pkg/security/apis/security"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/apis/admission"
)

func TestSCC(t *testing.T) {
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

	for _, tt := range []struct {
		name        string
		request     func() *admission.AdmissionReview
		wantMessage string
	}{
		{
			name: "modify customer scc allowed",
			request: func() *admission.AdmissionReview {
				return &admission.AdmissionReview{
					Request: &admission.AdmissionRequest{
						Name:   "customer",
						Object: &security.SecurityContextConstraints{},
					},
				}
			},
		},
		{
			name: "delete privileged scc not allowed",
			request: func() *admission.AdmissionReview {
				return &admission.AdmissionReview{
					Request: &admission.AdmissionRequest{
						Name:      "privileged",
						Operation: admission.Delete,
						Object:    sccs["privileged"],
					},
				}
			},
			wantMessage: `system SCC "privileged" may not be deleted`,
		},
		{
			name: "add user to privileged scc allowed",
			request: func() *admission.AdmissionReview {
				scc := sccs["privileged"].DeepCopy()
				scc.Labels["openshift.io/reconcile-protect"] = "true"
				scc.Users = append(scc.Users, "testuser")

				return &admission.AdmissionReview{
					Request: &admission.AdmissionRequest{
						Name:   "privileged",
						Object: scc,
					},
				}
			},
		},
		{
			name: "remove user from privileged scc not allowed",
			request: func() *admission.AdmissionReview {
				scc := sccs["privileged"].DeepCopy()
				scc.Labels["openshift.io/reconcile-protect"] = "true"
				scc.Users = scc.Users[1:]

				return &admission.AdmissionReview{
					Request: &admission.AdmissionRequest{
						Name:   "privileged",
						Object: scc,
					},
				}
			},
			wantMessage: "users: Required value: must include user system:admin",
		},
		{
			name: "add group to privileged scc allowed",
			request: func() *admission.AdmissionReview {
				scc := sccs["privileged"].DeepCopy()
				scc.Labels["openshift.io/reconcile-protect"] = "true"
				scc.Groups = append(scc.Groups, "testgroup")

				return &admission.AdmissionReview{
					Request: &admission.AdmissionRequest{
						Name:   "privileged",
						Object: scc,
					},
				}
			},
		},
		{
			name: "remove group from privileged scc not allowed",
			request: func() *admission.AdmissionReview {
				scc := sccs["privileged"].DeepCopy()
				scc.Labels["openshift.io/reconcile-protect"] = "true"
				scc.Groups = scc.Groups[1:]

				return &admission.AdmissionReview{
					Request: &admission.AdmissionRequest{
						Name:   "privileged",
						Object: scc,
					},
				}
			},
			wantMessage: "groups: Required value: must include group system:cluster-admins",
		},
		{
			name: "otherwise modify privileged scc not allowed",
			request: func() *admission.AdmissionReview {
				scc := sccs["privileged"].DeepCopy()
				scc.AllowPrivilegedContainer = false

				return &admission.AdmissionReview{
					Request: &admission.AdmissionRequest{
						Name:   "privileged",
						Object: scc,
					},
				}
			},
			wantMessage: `[]: Invalid value: "": may not modify fields other than users, groups and label labels.openshift.io/reconcile-protect`,
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

			response, err := doRequest(c, "https://server/sccs", request)
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
