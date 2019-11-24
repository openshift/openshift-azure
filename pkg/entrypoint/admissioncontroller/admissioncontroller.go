package admissioncontroller

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/ghodss/yaml"
	_ "github.com/openshift/origin/pkg/api/install"
	"github.com/openshift/origin/pkg/security/apis/security"
	"github.com/sirupsen/logrus"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster"
	"github.com/openshift/openshift-azure/pkg/sync"
	"github.com/openshift/openshift-azure/pkg/util/cloudprovider"
	"github.com/openshift/openshift-azure/pkg/util/configblob"
	"github.com/openshift/openshift-azure/pkg/util/log"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

var (
	codec = legacyscheme.Codecs.LegacyCodec(legacyscheme.Scheme.PrioritizedVersionsAllGroups()...)
)

type admissionController struct {
	l              net.Listener
	log            *logrus.Entry
	cs             *api.OpenShiftManagedCluster
	client         internalclientset.Interface
	imageWhitelist []*regexp.Regexp
	sccs           map[string]*security.SecurityContextConstraints
}

func start(cfg *cmdConfig) error {
	logrus.SetLevel(log.SanitizeLogLevel(cfg.LogLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log := logrus.NewEntry(logrus.StandardLogger())

	ac, err := newAdmissionController(context.Background(), log, cfg.configFile)
	if err != nil {
		return err
	}

	return ac.run()
}

func newAdmissionController(ctx context.Context, log *logrus.Entry, configFile string) (*admissionController, error) {
	/* #nosec - does this tool actually provide any value? */
	l, err := net.Listen("tcp", ":8443")
	if err != nil {
		return nil, err
	}

	cs, err := getCs(ctx, log)
	if err != nil {
		return nil, err
	}

	imageWhitelist, err := loadConfig(log, configFile)
	if err != nil {
		return nil, err
	}

	restconfig, err := managedcluster.RestConfigFromV1Config(cs.Config.MasterKubeconfig)
	if err != nil {
		return nil, err
	}

	client, err := internalclientset.NewForConfig(restconfig)
	if err != nil {
		return nil, err
	}

	sccs, err := readSyncPodSCCs(cs)
	if err != nil {
		return nil, err
	}

	return &admissionController{
		l:              l,
		log:            log,
		cs:             cs,
		client:         client,
		imageWhitelist: imageWhitelist,
		sccs:           sccs,
	}, nil
}

func getCs(ctx context.Context, log *logrus.Entry) (*api.OpenShiftManagedCluster, error) {
	cpc, err := cloudprovider.Load("_data/_out/azure.conf")
	if err != nil {
		return nil, err
	}

	bsc, err := configblob.GetService(ctx, log, cpc)
	if err != nil {
		return nil, err
	}

	c := bsc.GetContainerReference(cluster.ConfigContainerName)
	blob := c.GetBlobReference(cluster.MasterStartupBlobName)

	log.Print("reading config")
	rc, err := blob.Get(nil)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	// hack: we don't enrich cs like the sync pod does - this saves logic and
	// key accesses, but means calling sync.New() won't work without a bigger
	// refactor than I'm willing to do right now.

	var cs *api.OpenShiftManagedCluster
	return cs, json.NewDecoder(rc).Decode(&cs)
}

func loadConfig(log *logrus.Entry, filename string) ([]*regexp.Regexp, error) {
	var config struct {
		Whitelist []string `json:"whitelist,omitempty"`
	}

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}

	var whitelist []*regexp.Regexp
	for _, w := range config.Whitelist {
		rx, err := regexp.Compile(w)
		if err != nil {
			return nil, err
		}

		whitelist = append(whitelist, rx)
	}

	return whitelist, nil
}

func readSyncPodSCCs(cs *api.OpenShiftManagedCluster) (map[string]*security.SecurityContextConstraints, error) {
	assetNames, err := sync.AssetNames(cs)
	if err != nil {
		return nil, err
	}

	m := map[string]*security.SecurityContextConstraints{}
	for _, assetName := range assetNames {
		if !strings.HasPrefix(assetName, "SecurityContextConstraints.security.openshift.io/") {
			continue
		}

		a, err := sync.Asset(cs, assetName)
		if err != nil {
			return nil, err
		}

		scc := &security.SecurityContextConstraints{}
		_, _, err = codec.Decode(a, nil, scc)
		if err != nil {
			return nil, err
		}

		if scc.Labels == nil {
			scc.Labels = map[string]string{}
		}
		scc.Labels["azure.openshift.io/owned-by-sync-pod"] = "true"

		m[scc.Name] = scc
	}

	return m, nil
}

func (ac *admissionController) run() error {
	mux := &http.ServeMux{}
	mux.HandleFunc("/podwhitelist", ac.authenticated(ac.handleWhitelist))
	mux.HandleFunc("/sccs", ac.authenticated(ac.handleSCC))
	mux.HandleFunc("/healthz/ready", func(http.ResponseWriter, *http.Request) {})

	clientCAs := x509.NewCertPool()
	clientCAs.AddCert(ac.cs.Config.Certificates.Ca.Cert)

	tlsl := tls.NewListener(ac.l, &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{
				ac.cs.Config.Certificates.AroAdmissionController.Cert.Raw,
			},
			PrivateKey: ac.cs.Config.Certificates.AroAdmissionController.Key,
		}},
		ClientAuth: tls.VerifyClientCertIfGiven,
		ClientCAs:  clientCAs,
	})

	ac.log.Print("listening on :8443")
	return http.Serve(tlsl, &requestLogger{ac.log, mux})
}
