package admissioncontroller

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubernetes/pkg/apis/admission"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

type requestLogger struct {
	log *logrus.Entry
	http.Handler
}

func (rl *requestLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var subject string
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		subject = r.TLS.PeerCertificates[0].Subject.String()
	}

	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
	rl.Handler.ServeHTTP(rw, r)

	rl.log.WithFields(logrus.Fields{
		"requestRemoteAddr":             r.RemoteAddr,
		"requestPeerCertificateSubject": subject,
		"requestMethod":                 r.Method,
		"requestPath":                   r.URL.Path,
		"responseStatusCode":            rw.statusCode,
	}).Print()
}

func (ac *admissionController) authenticated(f func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil ||
			len(r.TLS.PeerCertificates) == 0 ||
			!bytes.Equal(r.TLS.PeerCertificates[0].Raw, ac.cs.Config.Certificates.AroAdmissionControllerClient.Cert.Raw) {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		f(w, r)
	}
}

func (ac *admissionController) getAdmissionRequest(r *http.Request) (*admission.AdmissionRequest, int) {
	if r.Method != http.MethodPost {
		return nil, http.StatusMethodNotAllowed
	}

	if r.Header.Get("Content-Type") != "application/json" {
		return nil, http.StatusUnsupportedMediaType
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, http.StatusBadRequest
	}

	review := &admission.AdmissionReview{}
	_, _, err = codec.Decode(b, nil, review)
	if err != nil {
		return nil, http.StatusBadRequest
	}

	req := review.Request
	if req == nil || req.UID == "" {
		return nil, http.StatusBadRequest
	}

	gvk := &schema.GroupVersionKind{Group: req.Kind.Group, Version: req.Kind.Version, Kind: req.Kind.Kind}
	o, _, err := codec.Decode(req.Object.(*runtime.Unknown).Raw, gvk, nil)
	if err != nil {
		return nil, http.StatusBadRequest
	}
	req.Object = o

	return req, 0
}

func (ac *admissionController) sendReview(w http.ResponseWriter, req *admission.AdmissionRequest, err error) {
	result := &metav1.Status{
		Status: metav1.StatusSuccess,
	}

	if err != nil {
		result = &metav1.Status{
			Status:  metav1.StatusFailure,
			Message: err.Error(),
		}
	}

	rev := &admission.AdmissionReview{
		Response: &admission.AdmissionResponse{
			UID:     req.UID,
			Allowed: result.Status == metav1.StatusSuccess,
			Result:  result,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	err = codec.Encode(rev, w)
	if err != nil {
		ac.log.Error(err)
	}
}
