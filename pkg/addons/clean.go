package addons

import (
	"regexp"

	"github.com/jim-minter/azure-helm/pkg/jsonpath"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

// Clean removes object entries which should not be persisted.
func Clean(o unstructured.Unstructured) {
	gk := o.GroupVersionKind().GroupKind()

	jsonpath.MustCompile("$.status").Delete(o.Object)

	switch gk.String() {
	case "DaemonSet.apps":
		jsonpath.MustCompile("$.metadata.annotations.'deprecated.daemonset.template.generation'").Delete(o.Object)
		cleanMetadata(jsonpath.MustCompile("$.spec.template").Get(o.Object)[0].(map[string]interface{}))

	case "Deployment.apps":
		jsonpath.MustCompile("$.metadata.annotations.'deployment.kubernetes.io/revision'").Delete(o.Object)
		cleanMetadata(jsonpath.MustCompile("$.spec.template").Get(o.Object)[0].(map[string]interface{}))

	case "DeploymentConfig.apps.openshift.io":
		cleanMetadata(jsonpath.MustCompile("$.spec.template").Get(o.Object)[0].(map[string]interface{}))

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

	case "Service":
		jsonpath.MustCompile("$.metadata.annotations.'service.alpha.openshift.io/serving-cert-signed-by'").Delete(o.Object)

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
		cleanMetadata(jsonpath.MustCompile("$.spec.template").Get(o.Object)[0].(map[string]interface{}))
	}

	cleanMetadata(o.Object)
}
