#!/bin/bash

# TODO: Should be obsoleted by "createorupdate.go -rm" once we start gathering artifacts in createorupdate

# teardown is collecting debug data and deleting all used resources
function teardown() {
  set +e
  mkdir -p "${HOME}"
  export HOME=/tmp/shared
  export DNS_DOMAIN=osadev.cloud
  export DNS_RESOURCEGROUP=dns
  export KUBECONFIG=/tmp/shared/_data/_out/admin.kubeconfig

  cp -r /tmp/shared/_data /go/src/github.com/openshift/openshift-azure/
  cd /go/src/github.com/openshift/openshift-azure/
  source /etc/azure/credentials/secret
  az login --service-principal -u ${AZURE_CLIENT_ID} -p ${AZURE_CLIENT_SECRET} --tenant ${AZURE_TENANT_ID} &>/dev/null

  # Gather artifacts
  oc get po --all-namespaces -o wide > /tmp/artifacts/pods
  oc get no -o wide > /tmp/artifacts/nodes
  oc get events --all-namespaces > /tmp/artifacts/events
  oc logs sync-master-000000 -n kube-system > /tmp/artifacts/sync.log
  oc logs master-api-master-000000 -n kube-system > /tmp/artifacts/api-master-000000.log
  oc logs master-api-master-000001 -n kube-system > /tmp/artifacts/api-master-000001.log
  oc logs master-api-master-000002 -n kube-system > /tmp/artifacts/api-master-000002.log
  oc logs master-etcd-master-000000 -n kube-system > /tmp/artifacts/etcd-master-000000.log
  oc logs master-etcd-master-000001 -n kube-system > /tmp/artifacts/etcd-master-000001.log
  oc logs master-etcd-master-000002 -n kube-system > /tmp/artifacts/etcd-master-000002.log
  cm_leader=$(oc get cm -n kube-system kube-controller-manager -o yaml | grep -o 00000[0-3])
  oc logs controllers-master-$cm_leader -n kube-system > /tmp/artifacts/controller-manager.log

  ./hack/delete.sh ${INSTANCE_PREFIX}
}

trap 'teardown' EXIT
trap 'kill $(jobs -p); exit 0' TERM

# teardown is triggered on file marker
for i in `seq 1 120`; do
  if [[ -f /tmp/shared/exit ]]; then
    exit 0
  fi
  sleep 60 & wait
done
