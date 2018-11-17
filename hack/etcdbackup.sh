#!/bin/bash -e

# This script is to create a manual etcd backup.
# automated (cron based) backups will continue to happen.

usage() {
    cat <<EOF >&2
usage:

$0 <backup-name>

Examples:
$0 before-upgrade-4.1.2

Note: if the name starts with "backup-" it will get pruned
EOF
    exit 1
}

if [[ -z "$KUBECONFIG" ]]; then
    echo error: must set KUBECONFIG
    exit 1
fi

if [[ -z "$ETCDBACKUP_IMAGE" ]]; then
    ETCDBACKUP_IMAGE=quay.io/openshift-on-azure/etcdbackup:latest
fi

name="$1"
if [[ -z "$name" ]]; then
    usage
fi

oc delete job etcd-manual-backup -n openshift-etcd || true
oc create -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: etcd-manual-backup
  namespace: openshift-etcd
spec:
  template:
    spec:
      nodeSelector:
        node-role.kubernetes.io/master: "true"
      serviceAccountName: etcd-backup
      restartPolicy: Never
      containers:
      - name: etcd-backup-hourly
        image: '$ETCDBACKUP_IMAGE'
        imagePullPolicy: Always
        args:
        - '-blobname=$name'
        - save
        volumeMounts:
        - name: azureconfig
          mountPath: /_data/_out
          readOnly: true
        - name: origin-master
          mountPath: /etc/origin/master
          readOnly: true
      volumes:
      - name: azureconfig
        hostPath:
          path: /etc/origin/cloudprovider
      - name: origin-master
        hostPath:
          path: /etc/origin/master
EOF
