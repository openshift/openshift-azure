apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: bundleinstances.automationbroker.io
spec:
  group: automationbroker.io
  names:
    kind: BundleInstance
    listKind: BundleInstanceList
    plural: bundleinstances
    singular: bundleinstance
  scope: Namespaced
  version: v1alpha1
