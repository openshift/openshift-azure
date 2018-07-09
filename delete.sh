#!/bin/bash -x

if [[ ! -e aks/admin.kubeconfig ]]; then
    echo error: aks/admin.kubeconfig must exist
    exit 1
fi

if ! az account show >/dev/null; then
    exit 1
fi

if [[ -z "$DNS_DOMAIN" ]]; then
    echo error: must set DNS_DOMAIN
    exit 1
fi

if [[ -z "$DNS_RESOURCEGROUP" ]]; then
    echo error: must set DNS_RESOURCEGROUP
    exit 1
fi

if [[ ! -e _data/manifest.yaml ]]; then
    echo error: _data/manifest.yaml must exist
    exit 1
fi

if [[ $# -ne 1 ]]; then
    echo usage: $0 resourcegroup
    exit 1
fi

RESOURCEGROUP=$1
PUBLICHOSTNAME=$(awk '/^  publicHostname:/ { print $2 }' <_data/manifest.yaml)

KUBECONFIG=aks/admin.kubeconfig helm delete --purge $RESOURCEGROUP >/dev/null

# k8s 1.10.3 seems to be very slow about removing terminating pods when their
# namespace is also terminating, so wait up.
while [[ $(KUBECONFIG=aks/admin.kubeconfig kubectl get pods -n $RESOURCEGROUP -o template --template '{{ len .items }}') -ne 0 ]]; do
    sleep 1
done

KUBECONFIG=aks/admin.kubeconfig kubectl delete namespace $RESOURCEGROUP

tools/dns.sh zone-delete $RESOURCEGROUP

tools/aad.sh app-delete $PUBLICHOSTNAME

rm -rf _data

az group delete -n $RESOURCEGROUP -y
