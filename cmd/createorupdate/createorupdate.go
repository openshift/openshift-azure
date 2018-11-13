package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/ghodss/yaml"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/openshift/openshift-azure/pkg/api"
	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	"github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	sdk "github.com/openshift/openshift-azure/pkg/util/azureclient/osa-go-sdk/services/containerservice/mgmt/2018-09-30-preview/containerservice"
)

const fakeRpAddr = "localhost:8080"

type server struct {
	// the server will not process more than a single
	// PUT request at all times.
	inProgress chan struct{}

	gc resources.GroupsClient

	sync.RWMutex
	state v20180930preview.ProvisioningState
	oc    *v20180930preview.OpenShiftManagedCluster

	log *logrus.Entry
}

func newServer(resourceGroup string) *server {
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	logger.SetLevel(logrus.DebugLevel)

	return &server{
		inProgress: make(chan struct{}, 1),
		log:        logrus.NewEntry(logger).WithFields(logrus.Fields{"resourceGroup": resourceGroup}),
	}
}

func (s *server) ListenAndServe() {
	// TODO: match the request path the real RP would use
	http.Handle("/", s)
	httpServer := &http.Server{Addr: fakeRpAddr}
	s.log.Infof("starting server on %s", fakeRpAddr)
	s.log.WithError(httpServer.ListenAndServe()).Warn("Server exited.")
}

// ServeHTTP handles an incoming request to the server.
func (s *server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	// validate the request
	ok := s.validate(w, req)
	if !ok {
		return
	}

	// process the request
	switch req.Method {
	case http.MethodDelete:
		s.handleDelete(w, req)
	case http.MethodGet:
		s.handleGet(w, req)
	case http.MethodPut:
		s.handlePut(w, req)
	}
}

func (s *server) validate(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPut && r.Method != http.MethodGet && r.Method != http.MethodDelete {
		resp := "405 Method not allowed"
		s.log.Debugf("%s: %s", r.Method, resp)
		http.Error(w, resp, http.StatusMethodNotAllowed)
		return false
	}

	if r.Method == http.MethodPut {
		select {
		case s.inProgress <- struct{}{}:
			// continue
		default:
			// did not get the lock
			resp := "423 Locked: Processing another in-flight request"
			s.log.Debug(resp)
			http.Error(w, resp, http.StatusLocked)
			return false
		}
	}
	return true
}

func (s *server) handleDelete(w http.ResponseWriter, req *http.Request) {
	// TODO: Get the azure credentials from the request headers
	authorizer, err := azureclient.NewAuthorizer(conf.ClientID, conf.ClientSecret, conf.TenantID)
	if err != nil {
		resp := "500 Internal Error: Failed to determine request credentials"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}
	// TODO: Determine subscription ID from the request path
	gc := resources.NewGroupsClient(conf.SubscriptionID)
	gc.Authorizer = authorizer

	resourceGroup := filepath.Base(req.URL.Path)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	s.log.Infof("deleting resource group %s", resourceGroup)

	future, err := gc.Delete(ctx, resourceGroup)
	if err != nil {
		resp := "500 Internal Error: Failed to delete resource group"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}
	if err := future.WaitForCompletionRef(ctx, gc.Client); err != nil {
		resp := "500 Internal Error: Failed to wait for resource group deletion"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}
	resp, err := future.Result(gc)
	if err != nil {
		resp := "500 Internal Error: Failed to get resource group deletion response"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}
	s.log.Infof("deleted resource group %s", resourceGroup)
	w.WriteHeader(resp.StatusCode)
}

func (s *server) handleGet(w http.ResponseWriter, req *http.Request) {
	s.reply(w, req)
}

func (s *server) handlePut(w http.ResponseWriter, req *http.Request) {
	defer func() {
		// drain once we are done processing this request
		<-s.inProgress
	}()

	b, err := ioutil.ReadAll(req.Body)
	if err != nil {
		resp := "400 Bad Request: Failed to read request body"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusBadRequest)
		return
	}

	var oc *v20180930preview.OpenShiftManagedCluster
	if err := yaml.Unmarshal(b, &oc); err != nil {
		resp := "400 Bad Request: Failed to unmarshal request"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusBadRequest)
		return
	}
	s.write(oc)

	// simulate Context with property bag
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()
	// TODO: Get the azure credentials from the request headers
	ctx = context.WithValue(ctx, api.ContextKeyClientID, conf.ClientID)
	ctx = context.WithValue(ctx, api.ContextKeyClientSecret, conf.ClientSecret)
	ctx = context.WithValue(ctx, api.ContextKeyTenantID, conf.TenantID)

	tc := api.TestConfig{
		RunningUnderTest:   os.Getenv("RUNNING_UNDER_TEST") != "",
		ImageResourceGroup: os.Getenv("IMAGE_RESOURCEGROUP"),
		ImageResourceName:  os.Getenv("IMAGE_RESOURCENAME"),
		DeployOS:           os.Getenv("DEPLOY_OS"),
		ImageOffer:         os.Getenv("IMAGE_OFFER"),
		ImageVersion:       os.Getenv("IMAGE_VERSION"),
		ORegURL:            os.Getenv("OREG_URL"),
	}

	config := &api.PluginConfig{
		SyncImage:       os.Getenv("SYNC_IMAGE"),
		LogBridgeImage:  os.Getenv("LOGBRIDGE_IMAGE"),
		AcceptLanguages: []string{"en-us"},
		TestConfig:      tc,
	}

	if currentState := s.readState(); string(currentState) == "" {
		s.writeState(v20180930preview.Creating)
	} else {
		// TODO: Need to separate between updates and upgrades
		s.writeState(v20180930preview.Updating)
	}

	if _, err := fakerp.CreateOrUpdate(ctx, oc, s.log, config); err != nil {
		s.writeState(v20180930preview.Failed)
		resp := "400 Bad Request: Failed to apply request"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusBadRequest)
		return
	}
	s.writeState(v20180930preview.Succeeded)
	s.reply(w, req)
}

func (s *server) write(oc *v20180930preview.OpenShiftManagedCluster) {
	s.Lock()
	defer s.Unlock()
	s.oc = oc
}

func (s *server) read() *v20180930preview.OpenShiftManagedCluster {
	s.RLock()
	defer s.RUnlock()
	return s.oc
}

func (s *server) writeState(state v20180930preview.ProvisioningState) {
	s.Lock()
	defer s.Unlock()
	s.state = state
}

func (s *server) readState() v20180930preview.ProvisioningState {
	s.RLock()
	defer s.RUnlock()
	return s.state
}

func (s *server) reply(w http.ResponseWriter, req *http.Request) {
	oc := s.read()
	if oc == nil {
		// This is a delete (trust me)
		// TODO: Need to model this better.
		return
	}
	oc.Properties.ProvisioningState = s.readState()
	res, err := json.Marshal(azureclient.ExternalToSdk(oc))
	if err != nil {
		resp := "500 Internal Server Error: Failed to marshal response"
		s.log.Debugf("%s: %v", resp, err)
		http.Error(w, resp, http.StatusInternalServerError)
		return
	}
	w.Write(res)
}

var conf config

type config struct {
	SubscriptionID   string `envconfig:"AZURE_SUBSCRIPTION_ID" required:"true"`
	TenantID         string `envconfig:"AZURE_TENANT_ID" required:"true"`
	ClientID         string `envconfig:"AZURE_CLIENT_ID" required:"true"`
	ClientSecret     string `envconfig:"AZURE_CLIENT_SECRET" required:"true"`
	AADClientID      string `envconfig:"AZURE_AAD_CLIENT_ID"`
	Region           string `envconfig:"AZURE_REGION"`
	DnsDomain        string `envconfig:"DNS_DOMAIN" required:"true"`
	DnsResourceGroup string `envconfig:"DNS_RESOURCEGROUP" required:"true"`
	ResourceGroup    string `envconfig:"RESOURCEGROUP" required:"true"`

	NoGroupTags      bool   `envconfig:"NOGROUPTAGS"`
	ResourceGroupTTL string `envconfig:"RESOURCEGROUP_TTL"`
}

func (c *config) init() error {
	supportedRegions := []string{"eastus", "westeurope", "australiasoutheast"}
	if c.Region == "" {
		// Randomly assign a supported region
		rand.Seed(time.Now().UTC().UnixNano())
		c.Region = supportedRegions[rand.Intn(3)]
		logrus.Infof("using randomly selected region %q", c.Region)
	}

	var supported bool
	for _, region := range supportedRegions {
		if c.Region == region {
			supported = true
		}
	}
	if !supported {
		return fmt.Errorf("%q is not a supported region (supported regions: %v)", c.Region, supportedRegions)
	}
	return nil
}

var (
	method   = flag.String("request", http.MethodPut, "Specify request to send to the OpenShift resource provider. Supported methods are PUT and DELETE.")
	useProd  = flag.Bool("use-prod", false, "If true, send the request to the production OpenShift resource provider.")
	manifest = flag.String("manifest", "_data/manifest.yaml", "Manifest to use for the initial request.")
	update   = flag.String("update", "", "If provided, use this manifest to make a follow-up request after the initial request succeeds.")
	cleanup  = flag.Bool("rm", false, "Delete the cluster once all other requests have completed successfully.")

	// timeouts
	rmTimeout     = flag.Duration("rm-timeout", 20*time.Minute, "Timeout of the cleanup request")
	timeout       = flag.Duration("timeout", 30*time.Minute, "Timeout of the initial request")
	updateTimeout = flag.Duration("update-timeout", 30*time.Minute, "Timeout of the update request")

	// exec hooks
	hook       = flag.String("exec", "", "Command to execute after the initial request to the RP has succeeded.")
	updateHook = flag.String("update-exec", "", "Command to execute after the update request to the RP has succeeded.")

	artifactDir        = flag.String("artifact-dir", "", "Directory to place artifacts before a cluster is deleted.")
	artifactKubeconfig = flag.String("artifact-kubeconfig", "", "Path to kubeconfig to use for gathering artifacts.")
)

func validate() error {
	switch strings.ToUpper(*method) {
	case http.MethodPut, http.MethodDelete:
	default:
		return fmt.Errorf("invalid request: %s, Supported methods are PUT and DELETE", strings.ToUpper(*method))
	}
	if *method == http.MethodDelete && *update != "" {
		return errors.New("cannot do an update when a DELETE is the initial request")
	}
	if *method == http.MethodDelete && *cleanup {
		return errors.New("cannot request a DELETE and -rm at the same time - use one of the two")
	}
	if *method == http.MethodDelete && (*hook != "" || *updateHook != "") {
		return errors.New("cannot request a DELETE and run an exec hook at the same time")
	}
	if *updateHook != "" && *update == "" {
		return errors.New("cannot exec an update hook when no update request is defined")
	}
	if (*artifactDir == "" && *artifactKubeconfig != "") || (*artifactDir != "" && *artifactKubeconfig == "") {
		return errors.New("both -artifact-dir and -artifact-kubeconfig need to be specified")
	}
	return nil
}

func delete(ctx context.Context, log *logrus.Entry, rpc sdk.OpenShiftManagedClustersClient) error {
	log.Info("deleting cluster")
	future, err := rpc.Delete(ctx, conf.ResourceGroup, conf.ResourceGroup)
	if err != nil {
		return err
	}
	if err := future.WaitForCompletionRef(ctx, rpc.Client); err != nil {
		return err
	}
	resp, err := future.Result(rpc)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s, expected 200 OK", resp.Status)
	}
	log.Info("deleted cluster")
	return nil
}

func createOrUpdate(ctx context.Context, log *logrus.Entry, rpc sdk.OpenShiftManagedClustersClient, manifest string) error {
	log.Info("creating/updating cluster")
	in, err := ioutil.ReadFile(manifest)
	if err != nil {
		return err
	}
	var oc sdk.OpenShiftManagedCluster
	if err := yaml.Unmarshal(in, &oc); err != nil {
		return err
	}
	future, err := rpc.CreateOrUpdate(ctx, conf.ResourceGroup, conf.ResourceGroup, oc)
	if err != nil {
		return err
	}
	if err := future.WaitForCompletionRef(ctx, rpc.Client); err != nil {
		return err
	}
	resp, err := future.Result(rpc)
	if err != nil {
		return err
	}
	out, err := yaml.Marshal(resp)
	if err != nil {
		return err
	}
	log.Info("created/updated cluster")
	return ioutil.WriteFile(manifest, out, 0666)
}

func execCommand(c string) error {
	args := strings.Split(c, " ")
	cmd := exec.Command(args[0], args[1:]...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s\n%v: %s", stdout.String(), err, stderr.String())
	}
	fmt.Println(stdout.String())
	return nil
}

func createResourceGroup(log *logrus.Entry) error {
	authorizer, err := azureclient.NewAuthorizer(conf.ClientID, conf.ClientSecret, conf.TenantID)
	if err != nil {
		return err
	}
	gc := resources.NewGroupsClient(conf.SubscriptionID)
	gc.Authorizer = authorizer

	if _, err := gc.Get(context.Background(), conf.ResourceGroup); err == nil {
		log.Infof("reusing existing resource group %s", conf.ResourceGroup)
		return nil
	}

	var tags map[string]*string
	if !conf.NoGroupTags {
		tags = make(map[string]*string)
		ttl, now := "76h", fmt.Sprintf("%d", time.Now().Unix())
		tags["now"] = &now
		tags["ttl"] = &ttl
		if conf.ResourceGroupTTL != "" {
			if _, err := time.ParseDuration(conf.ResourceGroupTTL); err != nil {
				return fmt.Errorf("invalid ttl provided: %q - %v", conf.ResourceGroupTTL, err)
			}
			tags["ttl"] = &conf.ResourceGroupTTL
		}
	}

	rg := resources.Group{Location: &conf.Region, Tags: tags}
	_, err = gc.CreateOrUpdate(context.Background(), conf.ResourceGroup, rg)
	return err
}

func gatherArtifacts(artifactDir, artifactKubeconfig string) error {
	config, err := clientcmd.BuildConfigFromFlags("", artifactKubeconfig)
	if err != nil {
		return err
	}
	kc, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	// gather node info
	if err := gatherNodes(kc, artifactDir); err != nil {
		return err
	}

	// gather pods from all namespaces
	// TODO: Ensure we don't leak any secrets. Fix either one of the following:
	// https://github.com/openshift/openshift-azure/issues/567
	// https://github.com/openshift/openshift-azure/issues/687
	// if err := gatherPods(kc, artifactDir); err != nil {
	//	return err
	// }

	// gather events from all namespaces
	if err := gatherEvents(kc, artifactDir); err != nil {
		return err
	}

	// gather control plane logs
	ns := "kube-system"
	if err := gatherLogs(kc, artifactDir, ns, "sync-master-000000"); err != nil {
		return err
	}
	// TODO: Get logs from the api server and etcd dynamically by using the master count
	if err := gatherLogs(kc, artifactDir, ns, "master-etcd-master-000000"); err != nil {
		return err
	}
	if err := gatherLogs(kc, artifactDir, ns, "master-etcd-master-000001"); err != nil {
		return err
	}
	if err := gatherLogs(kc, artifactDir, ns, "master-etcd-master-000002"); err != nil {
		return err
	}
	if err := gatherLogs(kc, artifactDir, ns, "master-api-master-000000"); err != nil {
		return err
	}
	if err := gatherLogs(kc, artifactDir, ns, "master-api-master-000001"); err != nil {
		return err
	}
	if err := gatherLogs(kc, artifactDir, ns, "master-api-master-000002"); err != nil {
		return err
	}
	// the controller manager uses leader election so only the leader can do writes.
	// Find out who is the leader and get its logs.
	cm, err := kc.CoreV1().ConfigMaps(ns).Get("kube-controller-manager", metav1.GetOptions{})
	if err != nil {
		return err
	}
	type leader struct {
		Holder string `json:"holderIdentity"`
	}
	var l leader
	if err := json.Unmarshal([]byte(cm.Annotations["control-plane.alpha.kubernetes.io/leader"]), &l); err != nil {
		return err
	}
	cmLeader := fmt.Sprintf("controllers-%s", strings.Split(l.Holder, "_")[0])
	return gatherLogs(kc, artifactDir, ns, cmLeader)
}

func gatherNodes(kc *kubernetes.Clientset, artifactDir string) error {
	nodeBuf := bytes.NewBuffer(nil)
	nodes, err := kc.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, node := range nodes.Items {
		b, err := yaml.Marshal(node)
		if err != nil {
			return err
		}
		if _, err := nodeBuf.Write(b); err != nil {
			return err
		}
		if _, err := nodeBuf.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return ioutil.WriteFile(filepath.Join(artifactDir, "nodes.yaml"), nodeBuf.Bytes(), 0777)
}

func gatherPods(kc *kubernetes.Clientset, artifactDir string) error {
	podBuf := bytes.NewBuffer(nil)
	pods, err := kc.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		b, err := yaml.Marshal(pod)
		if err != nil {
			return err
		}
		if _, err := podBuf.Write(b); err != nil {
			return err
		}
		if _, err := podBuf.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return ioutil.WriteFile(filepath.Join(artifactDir, "pods.yaml"), podBuf.Bytes(), 0777)
}

func gatherEvents(kc *kubernetes.Clientset, artifactDir string) error {
	eventBuf := bytes.NewBuffer(nil)
	events, err := kc.CoreV1().Events("").List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, event := range events.Items {
		b, err := yaml.Marshal(event)
		if err != nil {
			return err
		}
		if _, err := eventBuf.Write(b); err != nil {
			return err
		}
		if _, err := eventBuf.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return ioutil.WriteFile(filepath.Join(artifactDir, "events.yaml"), eventBuf.Bytes(), 0777)
}

func gatherLogs(kc *kubernetes.Clientset, artifactDir, ns, name string) error {
	log, err := kc.CoreV1().Pods(ns).GetLogs(name, &v1.PodLogOptions{}).DoRaw()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(artifactDir, fmt.Sprintf("%s_%s.log", ns, name)), log, 0777)
}

func main() {
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	log := logrus.NewEntry(logger)

	if err := envconfig.Process("", &conf); err != nil {
		log.Fatal(err)
	}
	if err := conf.init(); err != nil {
		log.Fatal(err)
	}
	log = logrus.NewEntry(logger).WithFields(logrus.Fields{"resourceGroup": conf.ResourceGroup})

	flag.Parse()
	if err := validate(); err != nil {
		log.Fatal(err)
	}

	if strings.ToUpper(*method) != http.MethodDelete {
		log.Infof("creating resource group %s", conf.ResourceGroup)
		if err := createResourceGroup(log); err != nil {
			log.Fatal(err)
		}
	}

	// simulate the RP
	if !*useProd {
		log.Info("starting the fake resource provider")
		s := newServer(conf.ResourceGroup)
		go s.ListenAndServe()
	}

	// setup the osa client
	rpURL := fmt.Sprintf("http://%s", fakeRpAddr)
	if *useProd {
		rpURL = sdk.DefaultBaseURI
	}
	rpc := sdk.NewOpenShiftManagedClustersClientWithBaseURI(rpURL, conf.SubscriptionID)
	authorizer, err := azureclient.NewAuthorizer(conf.ClientID, conf.ClientSecret, conf.TenantID)
	if err != nil {
		log.Fatal(err)
	}
	rpc.Authorizer = authorizer

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if strings.ToUpper(*method) == http.MethodDelete {
		if err := delete(ctx, log, rpc); err != nil {
			log.Fatal(err)
		}
		return
	}

	// if a cleanup is requested, do it unconditionally at the end
	if *cleanup {
		defer func() {
			delCtx, delCancel := context.WithTimeout(context.Background(), *rmTimeout)
			defer delCancel()
			if err := delete(delCtx, log, rpc); err != nil {
				log.Fatal(err)
			}
		}()
	}

	// simulate the API call to the RP
	if err := wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
		if err := createOrUpdate(ctx, log, rpc, *manifest); err != nil {
			if autoRestErr, ok := err.(autorest.DetailedError); ok {
				if urlErr, ok := autoRestErr.Original.(*url.Error); ok {
					if netErr, ok := urlErr.Err.(*net.OpError); ok {
						if sysErr, ok := netErr.Err.(*os.SyscallError); ok {
							if sysErr.Err == syscall.ECONNREFUSED {
								return false, nil
							}
						}
					}
				}
			}
			return false, err
		}
		return true, nil
	}); err != nil {
		log.Fatal(err)
	}

	if *hook != "" {
		if err := execCommand(*hook); err != nil {
			log.Fatal(err)
		}
	}

	// if an update is requested, do it
	if *update != "" {
		updateCtx, updateCancel := context.WithTimeout(context.Background(), *updateTimeout)
		defer updateCancel()
		if err := createOrUpdate(updateCtx, log, rpc, *update); err != nil {
			log.Fatal(err)
		}
	}

	if *updateHook != "" {
		if err := execCommand(*updateHook); err != nil {
			log.Fatal(err)
		}
	}

	if *artifactDir != "" {
		if err := gatherArtifacts(*artifactDir, *artifactKubeconfig); err != nil {
			log.Warnf("could not gather artifacts: %v", err)
		}
	}
}
