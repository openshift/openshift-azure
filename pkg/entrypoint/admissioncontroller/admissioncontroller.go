package admissioncontroller

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/openshift/origin/pkg/cmd/server/bootstrappolicy"
	"github.com/openshift/origin/pkg/security/apis/security"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"

	"github.com/openshift/openshift-azure/pkg/util/log"
)

var (
	serializer = kjson.NewSerializer(kjson.DefaultMetaFactory, legacyscheme.Scheme, legacyscheme.Scheme, false)
	codec      = legacyscheme.Codecs.CodecForVersions(nil, serializer, nil, runtime.InternalGroupVersioner)
)

type admissionControllerConfig struct {
	Whitelist []*regexp.Regexp
}

func (c *admissionControllerConfig) loadConfig(log *logrus.Entry) error {
	configFile, err := ioutil.ReadFile("/etc/aro-admission-controller/aro-admission-controller.yaml")
	if err != nil {
		log.Errorf("Error reading config file %s", err)
		return err
	}
	var cFile struct {
		Whitelist []string `json:"whitelist"`
	}
	err = yaml.Unmarshal(configFile, cFile)
	if err != nil {
		log.Errorf("Error unmarshalling config file %s", err)
		return err
	}
	for _, w := range cFile.Whitelist {
		rx, err := regexp.Compile(w)
		if err != nil {
			log.Errorf("Error interpreting config file %s", err)
			return err
		}
		c.Whitelist = append(c.Whitelist, rx)
	}
	return nil
}

func (ac *admissionController) handleHealthz(w http.ResponseWriter, r *http.Request) {
	return
}

type admissionController struct {
	log               *logrus.Entry
	client            internalclientset.Interface
	restricted        *security.SecurityContextConstraints
	whitelistedImages []*regexp.Regexp
	bootstrapSCCs     map[string]security.SecurityContextConstraints
	clientCAs         *x509.CertPool
}

func (ac *admissionController) getAdmissionReviewRequest(r *http.Request) (req *admissionv1beta1.AdmissionRequest, errorcode int) {
	ac.log.Debugf("New review request %s", r.RequestURI)
	if r.Method != http.MethodPost {
		return nil, http.StatusMethodNotAllowed
	}
	if r.Header.Get("Content-Type") != "application/json" {
		return nil, http.StatusUnsupportedMediaType
	}

	var reviewIncoming *admissionv1beta1.AdmissionReview
	err := json.NewDecoder(r.Body).Decode(&reviewIncoming)
	if err != nil {
		return nil, http.StatusBadRequest
	}
	req = reviewIncoming.Request
	gvk := schema.GroupVersionKind{Group: req.Kind.Group, Version: req.Kind.Version, Kind: req.Kind.Kind}
	o, _, err := codec.Decode(req.Object.Raw, &gvk, nil)
	if err != nil {
		ac.log.Errorf("Decode error:  %s", err)
		return nil, http.StatusBadRequest
	}
	// handle error case
	req.Object.Object = o
	return req, 0
}

func (ac *admissionController) ValidateRequest(r *http.Request, commonName string) error {
	subject := fmt.Sprintf("CN=%s", commonName)
	vOpts := x509.VerifyOptions{
		Roots:         ac.clientCAs,
		Intermediates: x509.NewCertPool(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	if len(r.TLS.PeerCertificates) == 0 {
		return fmt.Errorf("No certificates sent by client")
	}
	for _, cert := range r.TLS.PeerCertificates[1:] {
		vOpts.Intermediates.AddCert(cert)
	}
	chains, err := r.TLS.PeerCertificates[0].Verify(vOpts)
	if err != nil {
		return err
	}
	if len(chains) > 0 {
		if len(chains[0]) > 0 {
			if chains[0][0].Subject.String() == subject {
				return nil
			}
			return fmt.Errorf("Subject %s does not match %s", chains[0][0].Subject.String(), subject)
		}
	}
	return fmt.Errorf("No valid client certificates found")
}

func (ac *admissionController) AuthenticatedHandleWhitelist(w http.ResponseWriter, r *http.Request) {
	err := ac.ValidateRequest(r, "aro-admission-controller-client")
	if err != nil {
		//not authenticated with client TLS cert properly
		ac.log.Error(err)
		http.Error(w, http.StatusText(401), 401)
		return
	}
	ac.handleWhitelist(w, r)
}

func (ac *admissionController) AuthenticatedHandleSCC(w http.ResponseWriter, r *http.Request) {
	err := ac.ValidateRequest(r, "aro-admission-controller-client")
	if err != nil {
		//not authenticated with client TLS cert properly
		ac.log.Error(err)
		http.Error(w, http.StatusText(401), 401)
		return
	}
	ac.handleSCC(w, r)
}

func (ac *admissionController) run() error {
	ac.bootstrapSCCs = ac.InitProtectedSCCs()
	mux := &http.ServeMux{}
	mux.HandleFunc("/podwhitelist", ac.AuthenticatedHandleWhitelist)
	mux.HandleFunc("/sccs", ac.AuthenticatedHandleSCC)

	mux.HandleFunc("/healthz", ac.handleHealthz)
	mux.HandleFunc("/healthz/ready", ac.handleHealthz)

	CACert, err := ioutil.ReadFile("/etc/aro-admission-controller/ca.crt")
	if err != nil {
		ac.log.Fatal("Reading CA file failed: ", err)
	}
	clientCAs := x509.NewCertPool()
	clientCAs.AppendCertsFromPEM(CACert)
	tlsConfig := &tls.Config{
		ClientAuth:               tls.RequestClientCert,
		ClientCAs:                clientCAs,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
	}
	ac.log.Println("Aro Admission Controller starting.")
	s := http.Server{
		Addr:      ":8443",
		TLSConfig: tlsConfig,
		Handler:   mux,
	}
	ac.clientCAs = clientCAs
	err = s.ListenAndServeTLS("/etc/aro-admission-controller/aro-admission-controller.crt", "/etc/aro-admission-controller/aro-admission-controller.key")
	if err != nil {
		ac.log.Fatal("ListenAndServeTLS: ", err)
	}
	return err
}

func getRestrictedSCC() (*security.SecurityContextConstraints, error) {
	var restricted *security.SecurityContextConstraints

	groups, users := bootstrappolicy.GetBoostrapSCCAccess(bootstrappolicy.DefaultOpenShiftInfraNamespace)
	for _, scc := range bootstrappolicy.GetBootstrapSecurityContextConstraints(groups, users) {
		if scc.Name == bootstrappolicy.SecurityContextConstraintRestricted {
			restricted = scc
		}
	}
	if restricted == nil {
		return nil, fmt.Errorf("couldn't find restricted SCC in bootstrappolicy")
	}

	return restricted, nil
}

func start(cfg *cmdConfig) error {
	var c admissionControllerConfig

	logrus.SetLevel(log.SanitizeLogLevel(cfg.LogLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log := logrus.NewEntry(logrus.StandardLogger())

	err := c.loadConfig(log)
	if err != nil {
		return err
	}

	restricted, err := getRestrictedSCC()
	if err != nil {
		return err
	}

	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)

	restconfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return err
	}

	client, err := internalclientset.NewForConfig(restconfig)
	if err != nil {
		return err
	}

	ac := &admissionController{
		log:               log,
		client:            client,
		restricted:        restricted,
		whitelistedImages: c.Whitelist,
	}

	return ac.run()
}

func (ac *admissionController) sendResult(errs errors.Aggregate, w http.ResponseWriter, uid types.UID) {
	result := &metav1.Status{
		Status: metav1.StatusSuccess,
	}
	if errs != nil && len(errs.Errors()) > 0 {
		ac.log.Debugf("Found %d errs when validating", len(errs.Errors()))
		ac.log.Debugf("Error:%s", errs.Error())
		result = &metav1.Status{
			Status:  metav1.StatusFailure,
			Message: errs.Error(),
		}
	} else {
		ac.log.Debug("No errors found, approved")
	}
	rev := &admissionv1beta1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: admissionv1beta1.SchemeGroupVersion.String(),
			Kind:       "AdmissionReview",
		},
		Response: &admissionv1beta1.AdmissionResponse{
			UID:     uid,
			Allowed: result.Status == metav1.StatusSuccess,
			Result:  result,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(rev)
	if err != nil {
		ac.log.Errorf("Error encoding json: %s", err)
		return
	}
}
