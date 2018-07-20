package addons

import (
	"encoding/base64"
	"regexp"
	"unicode/utf8"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/jim-minter/azure-helm/pkg/jsonpath"
)

// cleanMetadata cleans an ObjectMeta structure
func cleanMetadata(obj map[string]interface{}) {
	metadataClean := []string{
		"$.metadata.annotations.'kubectl.kubernetes.io/last-applied-configuration'",
		"$.metadata.annotations.'openshift.io/generated-by'",
		"$.metadata.creationTimestamp",
		"$.metadata.generation",
		"$.metadata.resourceVersion",
		"$.metadata.selfLink",
		"$.metadata.uid",
	}
	for _, k := range metadataClean {
		jsonpath.MustCompile(k).Delete(obj)
	}

	path := jsonpath.MustCompile("$.metadata.annotations")
	annotations := path.Get(obj)
	if len(annotations) == 1 && len(annotations[0].(map[string]interface{})) == 0 {
		path.Delete(obj)
	}
}

// cleanPodTemplate cleans a pod template structure
func cleanPodTemplate(obj map[string]interface{}) {
	jsonpath.MustCompile("$.spec.initContainers.*.imagePullPolicy").Delete(obj)
	jsonpath.MustCompile("$.spec.containers.*.imagePullPolicy").Delete(obj)

	cleanMetadata(obj)
}

// convertSecretData converts data fields in a Secret to stringData fields
// wherever it can.
func convertSecretData(o unstructured.Unstructured) error {
	if _, found := o.Object["data"]; !found {
		return nil
	}

	data := o.Object["data"].(map[string]interface{})
	stringData := map[string]interface{}{}

	for k, v := range data {
		b, err := base64.StdEncoding.DecodeString(v.(string))
		if err != nil {
			return err
		}

		if utf8.Valid(b) {
			stringData[k] = string(b)
			delete(data, k)
		}
	}

	if len(stringData) > 0 {
		o.Object["stringData"] = stringData
	}
	if len(data) == 0 {
		delete(o.Object, "data")
	}

	return nil
}

// Clean removes object entries which should not be persisted.
func Clean(o unstructured.Unstructured) error {
	gk := o.GroupVersionKind().GroupKind()

	jsonpath.MustCompile("$.status").Delete(o.Object)

	switch gk.String() {
	case "DaemonSet.apps":
		jsonpath.MustCompile("$.metadata.annotations.'deprecated.daemonset.template.generation'").Delete(o.Object)
		cleanPodTemplate(jsonpath.MustCompile("$.spec.template").Get(o.Object)[0].(map[string]interface{}))

	case "Deployment.apps":
		jsonpath.MustCompile("$.metadata.annotations.'deployment.kubernetes.io/revision'").Delete(o.Object)
		cleanPodTemplate(jsonpath.MustCompile("$.spec.template").Get(o.Object)[0].(map[string]interface{}))

	case "DeploymentConfig.apps.openshift.io":
		cleanPodTemplate(jsonpath.MustCompile("$.spec.template").Get(o.Object)[0].(map[string]interface{}))

	case "ImageStream.image.openshift.io":
		jsonpath.MustCompile("$.metadata.annotations.'openshift.io/image.dockerRepositoryCheck'").Delete(o.Object)

	case "Namespace":
		// TODO: don't know exactly what we should do here.
		for _, k := range []string{
			"$.metadata.annotations.'openshift.io/sa.scc.mcs'",
			"$.metadata.annotations.'openshift.io/sa.scc.supplemental-groups'",
			"$.metadata.annotations.'openshift.io/sa.scc.uid-range'",
		} {
			jsonpath.MustCompile(k).Delete(o.Object)
		}

	case "Secret":
		if jsonpath.MustCompile("$.type").MustGetString(o.Object) == "kubernetes.io/service-account-token" {
			for _, k := range []string{
				"$.data",
				"$.metadata.annotations.'kubernetes.io/service-account.uid'",
			} {
				jsonpath.MustCompile(k).Delete(o.Object)
			}
		}

		err := convertSecretData(o)
		if err != nil {
			return err
		}

	case "Service":
		jsonpath.MustCompile("$.metadata.annotations.'service.alpha.openshift.io/serving-cert-signed-by'").Delete(o.Object)
		// for LoadBalancer type we need to preserve clusterIP
		if o.Object["spec"].(map[string]interface{})["type"] != "LoadBalancer" {
			jsonpath.MustCompile("$.spec.clusterIP").Delete(o.Object)
		}

	case "ServiceAccount":
		// TODO: the intention is to remove references to automatically created
		// secrets.
		for _, field := range []string{"imagePullSecrets", "secrets"} {
			var newRefs []interface{}
			for _, ref := range jsonpath.MustCompile("$." + field + ".*").Get(o.Object) {
				if !regexp.MustCompile("-[a-z0-9]{5}$").MatchString(jsonpath.MustCompile("$.name").MustGetString(ref)) {
					newRefs = append(newRefs, ref)
				}
			}
			if len(newRefs) > 0 {
				jsonpath.MustCompile("$."+field).Set(o.Object, newRefs)
			} else {
				jsonpath.MustCompile("$." + field).Delete(o.Object)
			}
		}

	case "StatefulSet.apps":
		cleanPodTemplate(jsonpath.MustCompile("$.spec.template").Get(o.Object)[0].(map[string]interface{}))
	}

	cleanMetadata(o.Object)

	return nil
}

// handleSpecialObjects manages special object migration during upgrade state
func handleSpecialObjects(existing, o unstructured.Unstructured) {

	switch existing.GetKind() {
	// Service type Loadbalancer
	case "Service":
		// copy existing clusterIP new object
		if existing.Object["spec"].(map[string]interface{})["type"] == "LoadBalancer" {
			o.Object["spec"].(map[string]interface{})["clusterIP"] = existing.Object["spec"].(map[string]interface{})["clusterIP"]
			o.Object["spec"].(map[string]interface{})["externalTrafficPolicy"] = existing.Object["spec"].(map[string]interface{})["externalTrafficPolicy"]
			o.Object["spec"].(map[string]interface{})["ports"] = existing.Object["spec"].(map[string]interface{})["ports"]
		}
	}
}
