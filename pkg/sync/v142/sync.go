package sync

//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -prefix data data/...
//go:generate gofmt -s -l -w bindata.go

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ghodss/yaml"
	authorizationv1 "github.com/openshift/client-go/authorization/clientset/versioned/typed/authorization/v1"
	"github.com/sirupsen/logrus"
	kapiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	dynamic "k8s.io/client-go/deprecated-dynamic"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/util/flowcontrol"
	kaggregator "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/jsonpath"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
	"github.com/openshift/openshift-azure/pkg/util/ready"
	"github.com/openshift/openshift-azure/pkg/util/roundtrippers"
	utilwait "github.com/openshift/openshift-azure/pkg/util/wait"
)

const (
	ownedBySyncPodLabelKey            = "azure.openshift.io/owned-by-sync-pod"
	optionallyApplySyncPodLabelKey    = "azure.openshift.io/sync-pod-optionally-apply"
	syncPodWaitForReadinessLabelKey   = "azure.openshift.io/sync-pod-wait-for-readiness"
	syncPodReadinessPathAnnotationKey = "azure.openshift.io/sync-pod-readiness-path"
	reconcileProtectAnnotationKey     = "openshift.io/reconcile-protect"
)

// unmarshal has to reimplement yaml.unmarshal because it universally mangles yaml
// integers into float64s, whereas the Kubernetes client library uses int64s
// wherever it can.  Such a difference can cause us to update objects when
// we don't actually need to.
func unmarshal(b []byte) (unstructured.Unstructured, error) {
	json, err := yaml.YAMLToJSON(b)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	var o unstructured.Unstructured
	_, _, err = unstructured.UnstructuredJSONScheme.Decode(json, nil, &o)
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	return o, nil
}

// ReadDB reads previously exported objects into a map via go-bindata as well as
// populating configuration items via translate().
func (s *sync) readDB() error {
	s.db = map[string]unstructured.Unstructured{}

	for _, asset := range AssetNames() {
		b, err := Asset(asset)
		if err != nil {
			return err
		}

		o, err := unmarshal(b)
		if err != nil {
			return err
		}

		o, err = translateAsset(o, s.cs)
		if err != nil {
			return err
		}

		s.db[keyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName())] = o
	}

	s.syncWorkloadsConfig()

	return nil
}

// syncWorkloadsConfig iterates over all workload controllers (deployments,
// daemonsets, statefulsets), walks their volumes, and updates their pod
// templates with annotations that include the hashes of the content for
// each configmap or secret.
func (s *sync) syncWorkloadsConfig() {
	// map config resources to their hashed content
	configToHash := make(map[string]string)
	for _, o := range s.db {
		gk := o.GroupVersionKind().GroupKind()

		if gk.String() != "Secret" &&
			gk.String() != "ConfigMap" {
			continue
		}

		configToHash[keyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName())] = getHash(&o)
	}

	// iterate over all workload controllers and add annotations with the hashes
	// of every config map or secret appropriately to force redeployments on config
	// updates.
	for _, o := range s.db {
		gk := o.GroupVersionKind().GroupKind()

		if gk.String() != "DaemonSet.apps" &&
			gk.String() != "Deployment.apps" &&
			gk.String() != "StatefulSet.apps" {
			continue
		}

		volumes := jsonpath.MustCompile("$.spec.template.spec.volumes.*").Get(o.Object)
		for _, v := range volumes {
			v := v.(map[string]interface{})

			if secretData, found := v["secret"]; found {
				secretName := jsonpath.MustCompile("$.secretName").MustGetString(secretData)
				key := fmt.Sprintf("checksum/secret-%s", secretName)
				secretKey := keyFunc(schema.GroupKind{Kind: "Secret"}, o.GetNamespace(), secretName)
				if hash, found := configToHash[secretKey]; found {
					setPodTemplateAnnotation(key, hash, o)
				}
			}

			if configMapData, found := v["configMap"]; found {
				configMapName := jsonpath.MustCompile("$.name").MustGetString(configMapData)
				key := fmt.Sprintf("checksum/configmap-%s", configMapName)
				configMapKey := keyFunc(schema.GroupKind{Kind: "ConfigMap"}, o.GetNamespace(), configMapName)
				if hash, found := configToHash[configMapKey]; found {
					setPodTemplateAnnotation(key, hash, o)
				}
			}
		}
	}
}

func getHash(o *unstructured.Unstructured) string {
	var content map[string]interface{}
	for _, v := range jsonpath.MustCompile("$.data").Get(o.Object) {
		content = v.(map[string]interface{})
	}
	for _, v := range jsonpath.MustCompile("$.stringData").Get(o.Object) {
		content = v.(map[string]interface{})
	}
	// sort config content appropriately
	var keys []string
	for key := range content {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	h := sha256.New()
	for _, key := range keys {
		fmt.Fprintf(h, "%s: %#v", key, content[key])
	}

	return hex.EncodeToString(h.Sum(nil))
}

// setPodTemplateAnnotation sets the provided key-value pair as an annotation
// inside the provided object's pod template.
func setPodTemplateAnnotation(key, value string, o unstructured.Unstructured) {
	annotations, _, _ := unstructured.NestedStringMap(o.Object, "spec", "template", "metadata", "annotations")
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[key] = value
	unstructured.SetNestedStringMap(o.Object, annotations, "spec", "template", "metadata", "annotations")
}

func (s *sync) calculateReadiness() (errs []error) {
	var keys []string
	for k := range s.db {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		o := s.db[k]

		if o.GetLabels()[syncPodWaitForReadinessLabelKey] == "false" {
			continue
		}

		gk := o.GroupVersionKind().GroupKind()

		switch gk.String() {
		case "DaemonSet.apps":
			ds, err := s.kc.AppsV1().DaemonSets(o.GetNamespace()).Get(o.GetName(), metav1.GetOptions{})
			if err != nil {
				errs = append(errs, err)
			} else if !ready.DaemonSetIsReady(ds) {
				errs = append(errs, fmt.Errorf("%s %s/%s is not ready: %d,%d/%d", gk.String(), o.GetNamespace(), o.GetName(), ds.Status.UpdatedNumberScheduled, ds.Status.NumberAvailable, ds.Status.DesiredNumberScheduled))
			}

		case "Deployment.apps":
			d, err := s.kc.AppsV1().Deployments(o.GetNamespace()).Get(o.GetName(), metav1.GetOptions{})
			if err != nil {
				errs = append(errs, err)
			} else if !ready.DeploymentIsReady(d) {
				specReplicas := int32(1)
				if d.Spec.Replicas != nil {
					specReplicas = *d.Spec.Replicas
				}

				errs = append(errs, fmt.Errorf("%s %s/%s is not ready: %d,%d/%d", gk.String(), o.GetNamespace(), o.GetName(), d.Status.UpdatedReplicas, d.Status.AvailableReplicas, specReplicas))
			}

		case "StatefulSet.apps":
			ss, err := s.kc.AppsV1().StatefulSets(o.GetNamespace()).Get(o.GetName(), metav1.GetOptions{})
			if err != nil {
				errs = append(errs, err)
			} else if !ready.StatefulSetIsReady(ss) {
				specReplicas := int32(1)
				if ss.Spec.Replicas != nil {
					specReplicas = *ss.Spec.Replicas
				}

				errs = append(errs, fmt.Errorf("%s %s/%s is not ready: %d,%d/%d", gk.String(), o.GetNamespace(), o.GetName(), ss.Status.UpdatedReplicas, ss.Status.ReadyReplicas, specReplicas))
			}

		case "Route.route.openshift.io":
			url := "https://" + jsonpath.MustCompile("$.spec.host").MustGetString(o.Object) + o.GetAnnotations()[syncPodReadinessPathAnnotationKey]
			cert := s.cs.Config.Certificates.Router.Certs
			cli := &http.Client{
				Transport: roundtrippers.HealthCheck(s.cs.Properties.RouterProfiles[0].FQDN, cert[len(cert)-1], s.cs.Location, nil),
				Timeout:   5 * time.Second,
			}
			resp, err := cli.Get(url)
			if err != nil {
				errs = append(errs, err)
			} else if resp.StatusCode != http.StatusOK {
				errs = append(errs, fmt.Errorf("%s %s/%s is not ready: %d", gk.String(), o.GetNamespace(), o.GetName(), resp.StatusCode))
			}
		}
	}

	return
}

// resource filters
var (
	crdFilter = func(o unstructured.Unstructured) bool {
		return o.GroupVersionKind().GroupKind() == schema.GroupKind{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition"}
	}
	nsFilter = func(o unstructured.Unstructured) bool {
		return o.GroupVersionKind().GroupKind() == schema.GroupKind{Kind: "Namespace"}
	}
	saFilter = func(o unstructured.Unstructured) bool {
		return o.GroupVersionKind().GroupKind() == schema.GroupKind{Kind: "ServiceAccount"}
	}
	cfgFilter = func(o unstructured.Unstructured) bool {
		return o.GroupVersionKind().GroupKind() == schema.GroupKind{Kind: "Secret"} ||
			o.GroupVersionKind().GroupKind() == schema.GroupKind{Kind: "ConfigMap"}
	}
	storageClassFilter = func(o unstructured.Unstructured) bool {
		return o.GroupVersionKind().GroupKind() == schema.GroupKind{Group: "storage.k8s.io", Kind: "StorageClass"}
	}
	serviceFilter = func(o unstructured.Unstructured) bool {
		return o.GroupVersionKind().Kind == "Service"
	}
	vwcFilter = func(o unstructured.Unstructured) bool {
		return o.GroupVersionKind().GroupKind() == schema.GroupKind{Group: "admissionregistration.k8s.io", Kind: "ValidatingWebhookConfiguration"}
	}
	roleFilter = func(o unstructured.Unstructured) bool {
		return o.GroupVersionKind().Kind == "ClusterRole" || o.GroupVersionKind().Kind == "Role"
	}
	roleBindingFilter = func(o unstructured.Unstructured) bool {
		return o.GroupVersionKind().Kind == "ClusterRoleBinding" || o.GroupVersionKind().Kind == "RoleBinding"
	}
	everythingElseFilter = func(o unstructured.Unstructured) bool {
		return !crdFilter(o) &&
			!nsFilter(o) &&
			!saFilter(o) &&
			!cfgFilter(o) &&
			!storageClassFilter(o) &&
			!scFilter(o) &&
			!monitoringCrdFilter(o) &&
			!vwcFilter(o) &&
			!serviceFilter(o) &&
			!roleFilter(o) &&
			!roleBindingFilter(o)
	}
	scFilter = func(o unstructured.Unstructured) bool {
		return o.GroupVersionKind().Group == "servicecatalog.k8s.io"
	}
	// targeted filter is used to target specific CRD - ServiceMonitor, which are managed not by sync pod
	monitoringCrdFilter = func(o unstructured.Unstructured) bool {
		return o.GroupVersionKind().Group == "monitoring.coreos.com"
	}
)

// writeDB uses the discovery and dynamic clients to synchronise an API server's
// objects with db.
// TODO: need to implement deleting objects which we don't want any more.
func (s *sync) writeDB() error {
	// openshift namespace is special. It hosts sharedResources.
	// Reconcile protection logic is isReconcileProtected func,
	// but we check if openshift namespace has protection annotation
	// to check if we need to apply sharedResourceFilter or not
	// default - resources are managed
	s.managedSharedResources = true
	namespace, err := s.kc.CoreV1().Namespaces().Get("openshift", metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if namespace != nil && strings.ToLower(namespace.GetAnnotations()[reconcileProtectAnnotationKey]) == "true" {
		s.managedSharedResources = false
	}

	// impose an order to improve debuggability.
	var keys []string
	for k := range s.db {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// crd needs to land early to get initialized
	s.log.Debug("applying crd resources")
	if err := s.applyResources(crdFilter, keys); err != nil {
		return err
	}
	// namespaces must exist before namespaced objects.
	s.log.Debug("applying ns resources")
	if err := s.applyResources(nsFilter, keys); err != nil {
		return err
	}
	// create serviceaccounts
	s.log.Debug("applying sa resources")
	if err := s.applyResources(saFilter, keys); err != nil {
		return err
	}
	// create all secrets and configmaps
	s.log.Debug("applying cfg resources")
	if err := s.applyResources(cfgFilter, keys); err != nil {
		return err
	}
	// default storage class must be created before PVCs as the admission controller is edge-triggered
	s.log.Debug("applying storageClass resources")
	if err := s.applyResources(storageClassFilter, keys); err != nil {
		return err
	}
	// create services before validatingwebhookconfigurations
	s.log.Debug("applying service resources")
	if err := s.applyResources(serviceFilter, keys); err != nil {
		return err
	}

	// create roles and clusterRoles
	s.log.Debug("applying role and clusterRole resources")
	if err := s.applyResources(roleFilter, keys); err != nil {
		return err
	}

	// create role and clusterRole bindings
	s.log.Debug("applying roleBinding and clusterRoleBinding resources")
	if err := s.applyResources(roleBindingFilter, keys); err != nil {
		return err
	}

	// wait for aro-admission-controller pods to be ready before validatingwebhookconfigurations
	s.log.Debug("waiting for admission controller pods")
	err = wait.PollImmediateInfinite(5*time.Second, func() (bool, error) {
		pods, err := s.kc.Core().Pods("kube-system").List(metav1.ListOptions{
			LabelSelector: "openshift.io/component=aro-admission-controller",
		})
		if err != nil {
			return false, err
		}
		var count int
		for _, pod := range pods.Items {
			if ready.PodIsReady(&pod) {
				count++
			}
		}
		return count == 3, nil
	})

	// create ValidatingWebhookConfiguration resources
	s.log.Debug("applying validatingwebhookconfiguration resources")
	if err := s.applyResources(vwcFilter, keys); err != nil {
		return err
	}

	// refresh dynamic client
	if err := s.updateDynamicClient(); err != nil {
		return err
	}

	// create all, except targeted CRDs resources
	s.log.Debug("applying other resources")
	if err := s.applyResources(everythingElseFilter, keys); err != nil {
		return err
	}

	// wait for the service catalog api extension to arrive. TODO: we should do
	// this dynamically, and should not PollInfinite.
	s.log.Debug("Waiting for the service catalog api to get aggregated")
	if err := wait.PollImmediateInfinite(time.Second,
		ready.CheckAPIServiceIsReady(s.ac.ApiregistrationV1().APIServices(), "v1beta1.servicecatalog.k8s.io"),
	); err != nil {
		return err
	}
	s.log.Debug("Service catalog api is aggregated")

	// refresh dynamic client
	if err := s.updateDynamicClient(); err != nil {
		return err
	}

	// now write the servicecatalog configurables.
	s.log.Debug("applying sc resources")
	if err := s.applyResources(scFilter, keys); err != nil {
		return err
	}

	// to speed up cluster startup time, we call this ready
	s.ready.Store(true)

	s.log.Debug("Waiting for the targeted CRDs to get ready")
	if err := wait.PollImmediateInfinite(time.Second,
		ready.CheckCustomResourceDefinitionIsReady(s.ae.ApiextensionsV1beta1().CustomResourceDefinitions(), "servicemonitors.monitoring.coreos.com"),
	); err != nil {
		return err
	}
	s.log.Debug("ServiceMonitors CRDs apis ready")

	// refresh dynamic client
	if err := s.updateDynamicClient(); err != nil {
		return err
	}

	// write all post boostrap objects depending on monitoring CRDs, managed by operators
	s.log.Debug("applying monitoring crd resources")
	return s.applyResources(monitoringCrdFilter, keys)
}

// Main loop
func (s *sync) Sync(ctx context.Context) error {
	transport, err := rest.TransportFor(s.restconfig)
	if err != nil {
		return err
	}

	_, err = utilwait.ForHTTPStatusOk(ctx, s.log, &http.Client{Transport: transport, Timeout: 10 * time.Second}, s.restconfig.Host+"/healthz", time.Second)
	if err != nil {
		return err
	}

	err = s.updateDynamicClient()
	if err != nil {
		return err
	}

	err = s.writeDB()
	if err != nil {
		return err
	}

	return s.deleteOrphans()
}

func (s *sync) ReadyHandler(w http.ResponseWriter, r *http.Request) {
	var errs []error

	if !s.ready.Load().(bool) {
		errs = []error{fmt.Errorf("sync pod has not completed first run")}
	} else {
		errs = s.calculateReadiness()
	}

	if len(errs) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-type", "text/plain")
	w.WriteHeader(http.StatusInternalServerError)
	for _, err := range errs {
		fmt.Fprintln(w, err)
	}
}

func (s *sync) Hash() ([]byte, error) {
	hash := sha256.New()

	// encoding/json sorts map keys as it encodes
	err := json.NewEncoder(hash).Encode(s.db)
	if err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

func (s *sync) PrintDB() error {
	// impose an order to improve debuggability.
	var keys []string
	for k := range s.db {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		b, err := yaml.Marshal(s.db[k].Object)
		if err != nil {
			return err
		}

		s.log.Info(string(b))
	}

	return nil
}

type sync struct {
	log *logrus.Entry

	kc    kubernetes.Interface
	cs    *api.OpenShiftManagedCluster
	db    map[string]unstructured.Unstructured
	ready atomic.Value

	restconfig *rest.Config
	ac         *kaggregator.Clientset
	ae         *kapiextensions.Clientset
	cli        *discovery.DiscoveryClient
	dyn        dynamic.ClientPool
	grs        []*restmapper.APIGroupResources
	auth       authorizationv1.ClusterRoleBindingsGetter

	managedSharedResources bool
}

func New(log *logrus.Entry, cs *api.OpenShiftManagedCluster, initClients bool) (*sync, error) {
	s := &sync{
		log: log,
		cs:  cs,
	}

	if initClients {
		var err error
		if cs.Properties.PrivateAPIServer {
			hostname := os.Getenv("masterNodeName")
			if hostname == "" {
				return nil, fmt.Errorf("could not read hostname from env[masterNodeName]")
			}
			cs.Config.AdminKubeconfig.Clusters[0].Cluster.Server = "https://" + hostname
		}
		s.restconfig, err = managedcluster.RestConfigFromV1Config(cs.Config.AdminKubeconfig)
		if err != nil {
			return nil, err
		}
		s.restconfig.RateLimiter = flowcontrol.NewFakeAlwaysRateLimiter()
		s.restconfig.WrapTransport = roundtrippers.NewRetryingRoundTripper(log, cs.Location, nil, true)

		s.kc, err = kubernetes.NewForConfig(s.restconfig)
		if err != nil {
			return nil, err
		}

		s.ac, err = kaggregator.NewForConfig(s.restconfig)
		if err != nil {
			return nil, err
		}

		s.ae, err = kapiextensions.NewForConfig(s.restconfig)
		if err != nil {
			return nil, err
		}

		s.cli, err = discovery.NewDiscoveryClientForConfig(s.restconfig)
		if err != nil {
			return nil, err
		}

		s.auth, err = authorizationv1.NewForConfig(s.restconfig)
		if err != nil {
			return nil, err
		}
	}

	s.ready.Store(false)

	err := s.readDB()
	if err != nil {
		return nil, err
	}

	return s, nil
}
