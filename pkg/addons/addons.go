package addons

//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -prefix data data/...
//go:generate gofmt -s -l -w bindata.go

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
	"time"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/jsonpath"
	"github.com/openshift/openshift-azure/pkg/util/ready"
)

const (
	ownedBySyncPodLabelKey          = "azure.openshift.io/owned-by-sync-pod"
	syncPodWaitForReadinessLabelKey = "azure.openshift.io/sync-pod-wait-for-readiness"
)

// Unmarshal has to reimplement yaml.Unmarshal because it universally mangles yaml
// integers into float64s, whereas the Kubernetes client library uses int64s
// wherever it can.  Such a difference can cause us to update objects when
// we don't actually need to.
func Unmarshal(b []byte) (unstructured.Unstructured, error) {
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
// populating configuration items via Translate().
func ReadDB(cs *api.OpenShiftManagedCluster) (map[string]unstructured.Unstructured, error) {
	db := map[string]unstructured.Unstructured{}

	for _, asset := range AssetNames() {
		b, err := Asset(asset)
		if err != nil {
			return nil, err
		}

		o, err := Unmarshal(b)
		if err != nil {
			return nil, err
		}

		o, err = translateAsset(o, cs)
		if err != nil {
			return nil, err
		}

		db[KeyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName())] = o
	}

	err := syncWorkloadsConfig(db)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// syncWorkloadsConfig iterates over all workload controllers (deployments,
// daemonsets, statefulsets), walks their volumes, and updates their pod
// templates with annotations that include the hashes of the content for
// each configmap or secret.
func syncWorkloadsConfig(db map[string]unstructured.Unstructured) error {
	// map config resources to their hashed content
	configToHash := make(map[string]string)
	for _, o := range db {
		gk := o.GroupVersionKind().GroupKind()

		if gk.String() != "Secret" &&
			gk.String() != "ConfigMap" {
			continue
		}

		configToHash[KeyFunc(o.GroupVersionKind().GroupKind(), o.GetNamespace(), o.GetName())] = getHash(&o)
	}

	// iterate over all workload controllers and add annotations with the hashes
	// of every config map or secret appropriately to force redeployments on config
	// updates.
	for _, o := range db {
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
				secretKey := KeyFunc(schema.GroupKind{Kind: "Secret"}, o.GetNamespace(), secretName)
				if hash, found := configToHash[secretKey]; found {
					setPodTemplateAnnotation(key, hash, o)
				}
			}

			if configMapData, found := v["configMap"]; found {
				configMapName := jsonpath.MustCompile("$.name").MustGetString(configMapData)
				key := fmt.Sprintf("checksum/configmap-%s", configMapName)
				configMapKey := KeyFunc(schema.GroupKind{Kind: "ConfigMap"}, o.GetNamespace(), configMapName)
				if hash, found := configToHash[configMapKey]; found {
					setPodTemplateAnnotation(key, hash, o)
				}
			}
		}
	}

	return nil
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

	return base64.StdEncoding.EncodeToString(h.Sum(nil))
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

func CalculateReadiness(kc kubernetes.Interface, db map[string]unstructured.Unstructured) (errs []error) {
	var keys []string
	for k := range db {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		o := db[k]

		if o.GetLabels()[syncPodWaitForReadinessLabelKey] == "false" {
			continue
		}

		gk := o.GroupVersionKind().GroupKind()

		switch gk.String() {
		case "DaemonSet.apps":
			ds, err := kc.AppsV1().DaemonSets(o.GetNamespace()).Get(o.GetName(), metav1.GetOptions{})
			if err != nil {
				errs = append(errs, err)
			} else if !ready.DaemonSetIsReady(ds) {
				errs = append(errs, fmt.Errorf("%s %s/%s is not ready: %d,%d/%d", gk.String(), o.GetNamespace(), o.GetName(), ds.Status.UpdatedNumberScheduled, ds.Status.NumberAvailable, ds.Status.DesiredNumberScheduled))
			}

		case "Deployment.apps":
			d, err := kc.AppsV1().Deployments(o.GetNamespace()).Get(o.GetName(), metav1.GetOptions{})
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
			ss, err := kc.AppsV1().StatefulSets(o.GetNamespace()).Get(o.GetName(), metav1.GetOptions{})
			if err != nil {
				errs = append(errs, err)
			} else if !ready.StatefulSetIsReady(ss) {
				specReplicas := int32(1)
				if ss.Spec.Replicas != nil {
					specReplicas = *ss.Spec.Replicas
				}

				errs = append(errs, fmt.Errorf("%s %s/%s is not ready: %d,%d/%d", gk.String(), o.GetNamespace(), o.GetName(), ss.Status.UpdatedReplicas, ss.Status.ReadyReplicas, specReplicas))
			}
		}
	}

	return
}

// resource filters
var (
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
	nonCRDFilter = func(o unstructured.Unstructured) bool {
		return o.GroupVersionKind().Group != "servicecatalog.k8s.io" &&
			o.GroupVersionKind().Group != "monitoring.coreos.com"
	}

	scFilter = func(o unstructured.Unstructured) bool {
		return o.GroupVersionKind().Group == "servicecatalog.k8s.io"
	}
	crdFilter = func(o unstructured.Unstructured) bool {
		return o.GroupVersionKind().GroupKind() == schema.GroupKind{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition"}
	}
	// targeted filter is used to target specific CRD - ServiceMonitor, which are managed not by sync pod
	monitoringCrdFilter = func(o unstructured.Unstructured) bool {
		return o.GroupVersionKind().GroupKind() == schema.GroupKind{Group: "monitoring.coreos.com", Kind: "ServiceMonitor"}
	}
	storageClassFilter = func(o unstructured.Unstructured) bool {
		return o.GroupVersionKind().GroupKind() == schema.GroupKind{Group: "storage.k8s.io", Kind: "StorageClass"}
	}
)

// writeDB uses the discovery and dynamic clients to synchronise an API server's
// objects with db.
// TODO: need to implement deleting objects which we don't want any more.
func writeDB(log *logrus.Entry, client *client, db map[string]unstructured.Unstructured) error {
	// impose an order to improve debuggability.
	var keys []string
	for k := range db {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// crd needs to land early to get initialized
	if err := client.ApplyResources(crdFilter, db, keys); err != nil {
		return err
	}
	// namespaces must exist before namespaced objects.
	if err := client.ApplyResources(nsFilter, db, keys); err != nil {
		return err
	}
	// create serviceaccounts
	if err := client.ApplyResources(saFilter, db, keys); err != nil {
		return err
	}
	// create all secrets and configmaps
	if err := client.ApplyResources(cfgFilter, db, keys); err != nil {
		return err
	}
	// default storage class must be created before PVCs as the admission controller is edge-triggered
	if err := client.ApplyResources(storageClassFilter, db, keys); err != nil {
		return err
	}

	// refresh dynamic client
	if err := client.UpdateDynamicClient(); err != nil {
		return err
	}

	// create all, except targeted CRDs resources
	if err := client.ApplyResources(nonCRDFilter, db, keys); err != nil {
		return err
	}

	// wait for the service catalog api extension to arrive. TODO: we should do
	// this dynamically, and should not PollInfinite.
	log.Debug("Waiting for the service catalog api to get aggregated")
	if err := wait.PollImmediateInfinite(time.Second,
		ready.CheckAPIServiceIsReady(client.ac.ApiregistrationV1().APIServices(), "v1beta1.servicecatalog.k8s.io"),
	); err != nil {
		return err
	}
	log.Debug("Service catalog api is aggregated")

	// refresh dynamic client
	if err := client.UpdateDynamicClient(); err != nil {
		return err
	}

	// now write the servicecatalog configurables.
	if err := client.ApplyResources(scFilter, db, keys); err != nil {
		return err
	}

	log.Debug("Waiting for the targeted CRDs to get ready")
	if err := wait.PollImmediateInfinite(time.Second,
		ready.CheckCustomResourceDefinitionIsReady(client.ae.ApiextensionsV1beta1().CustomResourceDefinitions(), "servicemonitors.monitoring.coreos.com"),
	); err != nil {
		return err
	}
	log.Debug("ServiceMonitors CRDs apis ready")

	// refresh dynamic client
	if err := client.UpdateDynamicClient(); err != nil {
		return err
	}

	// write all post boostrap objects depending on monitoring CRDs, managed by operators
	return client.ApplyResources(monitoringCrdFilter, db, keys)
}

func EnrichCSStorageAccountKeys(ctx context.Context, azs azureclient.AccountsClient, cs *api.OpenShiftManagedCluster) error {
	if cs.Config.RegistryStorageAccountKey == "" {
		key, err := azs.ListKeys(ctx, cs.Properties.AzProfile.ResourceGroup, cs.Config.RegistryStorageAccount)
		if err != nil {
			return err
		}
		cs.Config.RegistryStorageAccountKey = *(*key.Keys)[0].Value
	}

	if cs.Config.ConfigStorageAccount == "" {
		key, err := azs.ListKeys(ctx, cs.Properties.AzProfile.ResourceGroup, cs.Config.ConfigStorageAccount)
		if err != nil {
			return err
		}
		cs.Config.ConfigStorageAccountKey = *(*key.Keys)[0].Value
	}

	return nil
}

// Main loop
func Main(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster, db map[string]unstructured.Unstructured) error {
	client, err := newClient(ctx, log, cs)
	if err != nil {
		return err
	}

	err = writeDB(log, client, db)
	if err != nil {
		return err
	}

	return client.DeleteOrphans(db)
}
