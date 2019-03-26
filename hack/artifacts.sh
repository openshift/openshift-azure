#!/bin/bash -ex

mkdir -p /tmp/artifacts

export KUBECONFIG=$PWD/_data/_out/admin.kubeconfig
oc get po --all-namespaces -o wide > /tmp/artifacts/pods
oc get deployments --all-namespaces -o wide > /tmp/artifacts/deployments
oc get statefulsets --all-namespaces -o wide > /tmp/artifacts/statefulsets
oc get daemonsets --all-namespaces -o wide > /tmp/artifacts/daemonsets
oc get no -o wide > /tmp/artifacts/nodes
oc get events --all-namespaces > /tmp/artifacts/events
oc logs deployment/sync -n kube-system > /tmp/artifacts/sync.log
oc logs master-api-master-000000 -n kube-system > /tmp/artifacts/api-master-000000.log
oc logs master-api-master-000001 -n kube-system > /tmp/artifacts/api-master-000001.log
oc logs master-api-master-000002 -n kube-system > /tmp/artifacts/api-master-000002.log
oc logs master-etcd-master-000000 -n kube-system > /tmp/artifacts/etcd-master-000000.log
oc logs master-etcd-master-000001 -n kube-system > /tmp/artifacts/etcd-master-000001.log
oc logs master-etcd-master-000002 -n kube-system > /tmp/artifacts/etcd-master-000002.log
cm_leader=$(oc get cm -n kube-system kube-controller-manager -o yaml | grep -o 00000[0-3])
oc logs controllers-master-$cm_leader -n kube-system > /tmp/artifacts/controller-manager.log
oc logs deploy/prometheus-operator -n openshift-monitoring > /tmp/artifacts/prometheus-operator.log
oc exec -n openshift-monitoring -it $(oc get po -n openshift-monitoring -l k8s-app=prometheus-operator --no-headers | awk '{print $1}') wget -- -O - localhost:8080/debug/pprof/goroutine?debug=2 >  /tmp/artifacts/prometheus-operator-pprof-goroutine
oc logs deploy/cluster-monitoring-operator -n openshift-monitoring > /tmp/artifacts/cluster-monitoring-operator.log
