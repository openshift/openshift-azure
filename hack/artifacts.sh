#!/bin/bash

# TODO: we should consider dropping most of this in favour of using Geneva more.

if [[ ! -e $PWD/_data/_out/admin.kubeconfig ]]; then
    echo "admin.kubeconfig not found, exiting"
    exit 0
fi

if [[ -z "$ARTIFACTS" ]]; then
    exit 0
fi

mkdir -p "$ARTIFACTS"

export KUBECONFIG=$PWD/_data/_out/admin.kubeconfig

for ((i=0; i<3; i++)); do
    oc logs -n kube-system master-api-master-00000$i >"$ARTIFACTS/api-master-00000$i.log"
    oc logs -n kube-system master-etcd-master-00000$i >"$ARTIFACTS/etcd-master-00000$i.log"
    true
done

cm_leader=$(oc get configmap -n kube-system kube-controller-manager -o yaml | grep -o 00000[0-3])
oc logs controllers-master-$cm_leader -n kube-system >"$ARTIFACTS/controller-manager.log"

for deployment in \
        kube-system/sync \
        openshift-monitoring/cluster-monitoring-operator \
        openshift-monitoring/prometheus-operator \
        ; do
	namespace="${deployment%%/*}"
	name="${deployment##*/}"

    oc logs -n "$namespace" "deployment/$name" >"$ARTIFACTS/$name.log"
done

for kind in \
        daemonsets \
        deployments \
        events \
        nodes \
        pods \
        statefulsets \
        ; do
    oc get "$kind" --all-namespaces -o wide >"$ARTIFACTS/$kind"
done

for node in $(oc get nodes -o jsonpath='{.items[*].metadata.name}'); do
    oc get --raw "/api/v1/nodes/$node/proxy/debug/pprof/goroutine?debug=2" >"$ARTIFACTS/$node-goroutines"
    hack/scp.sh "$node":'/var/crash/*/vmcore-dmesg.txt' "$ARTIFACTS/$node-vmcore-dmesg.txt"
done

po_pod=$(oc get pod -n openshift-monitoring -l k8s-app=prometheus-operator -o jsonpath='{.items[0].metadata.name}')
oc get --raw "/api/v1/namespaces/openshift-monitoring/pods/$po_pod/proxy/debug/pprof/goroutine?debug=2"
