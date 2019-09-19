package openshift

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

var (
	kafkaClusterGVK = schema.GroupVersionKind{
		Group:   "kafka.strimzi.io",
		Version: "v1alpha1",
		Kind:    "Kafka",
	}
)

// GetKafka return the kafka CR with the given name and namespace
func (cli *Client) GetKafka(name, namespace string) (*unstructured.Unstructured, error) {
	groupresources, err := discovery.GetAPIGroupResources(cli.Discovery)
	if err != nil {
		return nil, err
	}

	rmapper := discovery.NewRESTMapper(groupresources, meta.InterfacesForUnstructured)
	dynamicclientpool := dynamic.NewClientPool(cli.config, rmapper, dynamic.LegacyAPIPathResolverFunc)

	dynamicclient, err := dynamicclientpool.ClientForGroupVersionKind(kafkaClusterGVK)
	if err != nil {
		return nil, err
	}

	restmapping, err := rmapper.RESTMapping(kafkaClusterGVK.GroupKind(), kafkaClusterGVK.Version)
	if err != nil {
		return nil, err
	}

	apiresource := &metav1.APIResource{
		Name:       restmapping.Resource,
		Namespaced: restmapping.Scope.Name() == meta.RESTScopeNameNamespace,
	}

	return dynamicclient.Resource(apiresource, namespace).Get(name, metav1.GetOptions{})
}

// DeleteKafka delete the cluster
func (cli *Client) DeleteKafka(name, namespace string) error {
	groupresources, err := discovery.GetAPIGroupResources(cli.Discovery)
	if err != nil {
		return err
	}

	rmapper := discovery.NewRESTMapper(groupresources, meta.InterfacesForUnstructured)
	dynamicclientpool := dynamic.NewClientPool(cli.config, rmapper, dynamic.LegacyAPIPathResolverFunc)

	dynamicclient, err := dynamicclientpool.ClientForGroupVersionKind(kafkaClusterGVK)
	if err != nil {
		return err
	}

	restmapping, err := rmapper.RESTMapping(kafkaClusterGVK.GroupKind(), kafkaClusterGVK.Version)
	if err != nil {
		return err
	}

	apiresource := &metav1.APIResource{
		Name:       restmapping.Resource,
		Namespaced: restmapping.Scope.Name() == meta.RESTScopeNameNamespace,
	}

	return dynamicclient.Resource(apiresource, namespace).Delete(name, nil)
}
