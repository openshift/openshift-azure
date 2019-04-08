#!/bin/bash -x

# TODO: we should consider dropping most of this in favour of using Geneva more.

if [[ -z "$ARTIFACT_DIR" ]]; then
    exit 0
fi

if [[ ! -e $PWD/_data/_out/admin.kubeconfig ]]; then
    echo "admin.kubeconfig not found, exiting"
    exit 0
fi

mkdir -p "$ARTIFACT_DIR"

export KUBECONFIG=$PWD/_data/_out/admin.kubeconfig

for ((i=0; i<3; i++)); do
    oc logs -n kube-system master-api-master-00000$i >"$ARTIFACT_DIR/api-master-00000$i.log"
    oc logs -n kube-system master-etcd-master-00000$i >"$ARTIFACT_DIR/etcd-master-00000$i.log"
    true
done

cm_leader=$(oc get configmap -n kube-system kube-controller-manager -o yaml | grep -o 00000[0-3])
oc logs controllers-master-$cm_leader -n kube-system >"$ARTIFACT_DIR/controller-manager.log"

for deployment in \
        kube-system/sync \
        openshift-monitoring/cluster-monitoring-operator \
        openshift-monitoring/prometheus-operator \
        ; do
	namespace="${deployment%%/*}"
	name="${deployment##*/}"

    oc logs -n "$namespace" "deployment/$name" >"$ARTIFACT_DIR/$name.log"
done

for kind in \
        daemonsets \
        deployments \
        events \
        nodes \
        pods \
        statefulsets \
        ; do
    oc get "$kind" --all-namespaces -o wide >"$ARTIFACT_DIR/$kind"
done

for node in $(oc get nodes -o jsonpath='{.items[*].metadata.name}'); do
    oc get --raw "/api/v1/nodes/$node/proxy/debug/pprof/goroutine?debug=2" >"$ARTIFACT_DIR/$node-goroutines"
done

po_pod=$(oc get pod -n openshift-monitoring -l k8s-app=prometheus-operator -o jsonpath='{.items[0].metadata.name}')
oc get --raw "/api/v1/namespaces/openshift-monitoring/pods/$po_pod/proxy/debug/pprof/goroutine?debug=2"
