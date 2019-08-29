package openshift

import (
	"time"

	templatev1 "github.com/openshift/api/template/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/wait"
	deprecated_dynamic "k8s.io/client-go/deprecated-dynamic"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/restmapper"

	"github.com/openshift/openshift-azure/pkg/util/ready"
)

func init() {
	// The Kubernetes Go client (nested within the OpenShift Go client)
	// automatically registers its types in scheme.Scheme, however the
	// additional OpenShift types must be registered manually.  AddToScheme
	// registers the API group types (e.g. route.openshift.io/v1, Route) only.

	if err := templatev1.Install(scheme.Scheme); err != nil {
		panic(err)
	}
}

// InstantiateTemplate gets an openshift template and instantiates it
func (cli *Client) InstantiateTemplate(srcTemplateName, dstNamespace string) error {
	template, err := cli.TemplateV1.Templates("openshift").Get(srcTemplateName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	_, err = cli.TemplateV1.TemplateInstances(dstNamespace).Create(
		&templatev1.TemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: dstNamespace,
			},
			Spec: templatev1.TemplateInstanceSpec{
				Template: *template,
			},
		})
	if err != nil {
		return err
	}

	return wait.PollImmediate(2*time.Second, 20*time.Minute, ready.CheckTemplateInstanceIsReady(cli.TemplateV1.TemplateInstances(dstNamespace), dstNamespace))
}

// InstantiateTemplateFromBytes instantiates an openshift template from a byte slice
func (cli *Client) InstantiateTemplateFromBytes(yamldata []byte, dstNamespace string, parameters map[string]string) error {
	groupresources, err := restmapper.GetAPIGroupResources(cli.Discovery)
	if err != nil {
		return err
	}

	rmapper := restmapper.NewDiscoveryRESTMapper(groupresources)
	dynamicclientpool := deprecated_dynamic.NewClientPool(cli.config, rmapper, deprecated_dynamic.LegacyAPIPathResolverFunc)

	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	template := &templatev1.Template{}
	_, _, err = s.Decode(yamldata, nil, template)
	if err != nil {
		return err
	}

	for k, v := range parameters {
		for i, param := range template.Parameters {
			if param.Name == k {
				template.Parameters[i].Value = v
			}
		}
	}

	err = cli.TemplateV1.RESTClient().Post().
		Namespace(dstNamespace).
		Resource("processedTemplates").
		Body(template).
		Do().
		Into(template)
	if err != nil {
		return err
	}

	for i, o := range template.Objects {
		object, _, err := unstructured.UnstructuredJSONScheme.Decode(o.Raw, nil, nil)
		if err != nil {
			return err
		}

		template.Objects[i] = runtime.RawExtension{Object: object}
	}

	for _, o := range template.Objects {
		o := o.Object.(*unstructured.Unstructured)
		gvk := o.GroupVersionKind()
		rmapper, err := rmapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return err
		}

		dynamicclient, err := dynamicclientpool.ClientForGroupVersionKind(gvk)
		if err != nil {
			return err
		}

		apiresource := &metav1.APIResource{
			Name:       rmapper.Resource.Resource,
			Namespaced: rmapper.Scope.Name() == meta.RESTScopeNameNamespace,
		}
		if apiresource.Namespaced {
			o.SetNamespace(dstNamespace)
		}

		_, err = dynamicclient.Resource(apiresource, o.GetNamespace()).Create(o)
		if err != nil {
			return err
		}
	}

	return nil
}
