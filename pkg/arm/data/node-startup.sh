#!/bin/bash -ex

# TODO: /etc/dnsmasq.d/origin-upstream-dns.conf is currently hardcoded; it
# probably shouldn't be

SERVICE_TYPE=origin
if [ -f "/etc/sysconfig/atomic-openshift-node" ]; then
    SERVICE_TYPE=atomic-openshift
fi

# remove registry certificate softlink from docker
unlink /etc/docker/certs.d/registry.access.redhat.com/redhat-ca.crt

if ! grep /var/lib/docker /etc/fstab; then
  systemctl stop docker.service
  mkfs.xfs -f /dev/disk/azure/resource-part1
  echo '/dev/disk/azure/resource-part1  /var/lib/docker  xfs  grpquota  0 0' >>/etc/fstab
  mount /var/lib/docker
  restorecon -R /var/lib/docker
{{- if eq .Extra.Role "infra" }}
  cat >/etc/docker/daemon.json <<'EOF'
{
  "log-driver": "journald"
}
EOF
{{- end }}
  systemctl start docker.service
fi

# file should be /root/.docker/config.json, but actually need it in
# /var/lib/origin thanks to https://github.com/kubernetes/kubernetes/issues/45487
DPATH=/var/lib/origin/.docker
mkdir -p $DPATH
base64 -d <<< {{ .Config.Images.ImagePullSecret | Base64Encode }} >${DPATH}/config.json; chmod 0600 ${DPATH}/config.json

echo 'BOOTSTRAP_CONFIG_NAME=node-config-{{ .Extra.Role }}' >>/etc/sysconfig/${SERVICE_TYPE}-node

sed -i -e "s#DEBUG_LOGLEVEL=.*#DEBUG_LOGLEVEL={{ .Config.ComponentLogLevel.Node }}#" /etc/sysconfig/${SERVICE_TYPE}-node

rm -rf /etc/etcd/* /etc/origin/master/*

base64 -d <<< {{ Base64Encode (YamlMarshal .Config.NodeBootstrapKubeconfig) }} >/etc/origin/node/bootstrap.kubeconfig; chmod 0600 /etc/origin/node/bootstrap.kubeconfig
base64 -d <<< {{ Base64Encode (YamlMarshal .Config.SDNKubeconfig) }} >/etc/origin/node/sdn.kubeconfig; chmod 0600 /etc/origin/node/sdn.kubeconfig
base64 -d <<< {{ Base64Encode (CertAsBytes .Config.Certificates.NodeBootstrap.Cert) }} >/etc/origin/node/node-bootstrapper.crt
base64 -d <<< {{ Base64Encode (PrivateKeyAsBytes .Config.Certificates.NodeBootstrap.Key) }} >/etc/origin/node/node-bootstrapper.key; chmod 0600 /etc/origin/node/node-bootstrapper.key

base64 -d <<< {{ Base64Encode (CertAsBytes .Config.Certificates.Ca.Cert) }} >/etc/origin/node/ca.crt
cp /etc/origin/node/ca.crt /etc/pki/ca-trust/source/anchors/openshift-ca.crt
update-ca-trust

echo 'nameserver 168.63.129.16' >/etc/origin/node/resolv.conf
mkdir -p /etc/origin/cloudprovider

cat >/etc/origin/cloudprovider/azure.conf <<'EOF'; chmod 0600 /etc/origin/cloudprovider/azure.conf
{{ .Derived.WorkerCloudProviderConf .ContainerService | String }}
EOF

mkdir -p /etc/origin/node/pods
cat >/etc/origin/node/pods/ovs.yaml <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  annotations:
    scheduler.alpha.kubernetes.io/critical-pod: ''
  labels:
    app: ovs
    component: network
    openshift.io/component: network
    type: infra
  name: ovs
  namespace: openshift-sdn
spec:
  containers:
  - command:
    - /bin/bash
    - -c
    - |
      #!/bin/bash
      set -euo pipefail

      # if another process is listening on the cni-server socket, wait until it exits
      trap 'kill $(jobs -p); exit 0' TERM
      retries=0
      while true; do
        if /usr/share/openvswitch/scripts/ovs-ctl status &>/dev/null; then
          echo "warning: Another process is currently managing OVS, waiting 15s ..." 2>&1
          sleep 15 & wait
          (( retries += 1 ))
        else
          break
        fi
        if [[ "${retries}" -gt 40 ]]; then
          echo "error: Another process is currently managing OVS, exiting" 2>&1
          exit 1
        fi
      done

      # launch OVS
      function quit {
          /usr/share/openvswitch/scripts/ovs-ctl stop
          exit 0
      }
      trap quit SIGTERM
      /usr/share/openvswitch/scripts/ovs-ctl start --no-ovs-vswitchd --system-id=random

      # Restrict the number of pthreads ovs-vswitchd creates to reduce the
      # amount of RSS it uses on hosts with many cores
      # https://bugzilla.redhat.com/show_bug.cgi?id=1571379
      # https://bugzilla.redhat.com/show_bug.cgi?id=1572797
      if [[ `nproc` -gt 12 ]]; then
          ovs-vsctl --no-wait set Open_vSwitch . other_config:n-revalidator-threads=4
          ovs-vsctl --no-wait set Open_vSwitch . other_config:n-handler-threads=10
      fi
      /usr/share/openvswitch/scripts/ovs-ctl start --no-ovsdb-server --system-id=random
      while true; do sleep 5; done
    image: {{ .Config.Images.Node | quote }}
    name: openvswitch
    resources:
      limits:
        cpu: 200m
        memory: 400Mi
      requests:
        cpu: 10m
        memory: 100Mi
    securityContext:
      privileged: true
      runAsUser: 0
    volumeMounts:
    - mountPath: /lib/modules
      name: host-modules
      readOnly: true
    - mountPath: /run/openvswitch
      name: host-run-ovs
    - mountPath: /var/run/openvswitch
      name: host-run-ovs
    - mountPath: /sys
      name: host-sys
      readOnly: true
    - mountPath: /etc/openvswitch
      name: host-config-openvswitch
  hostNetwork: true
  hostPID: true
  priorityClassName: system-node-critical
  volumes:
  - hostPath:
      path: /lib/modules
    name: host-modules
  - hostPath:
      path: /run/openvswitch
    name: host-run-ovs
  - hostPath:
      path: /sys
    name: host-sys
  - hostPath:
      path: /etc/origin/openvswitch
    name: host-config-openvswitch
EOF

cat >/etc/origin/node/pods/sdn.yaml <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  annotations:
    scheduler.alpha.kubernetes.io/critical-pod: ""
  labels:
    app: sdn
    component: network
    openshift.io/component: network
    type: infra
  name: sdn
  namespace: openshift-sdn
spec:
  containers:
  - command:
    - /bin/bash
    - -c
    - |
      #!/bin/bash
      set -euo pipefail

      # if another process is listening on the cni-server socket, wait until it exits
      trap 'kill $(jobs -p); exit 0' TERM
      retries=0
      while true; do
        if echo 'test' | socat - UNIX-CONNECT:/var/run/openshift-sdn/cni-server.sock >/dev/null; then
          echo "warning: Another process is currently listening on the CNI socket, waiting 15s ..." 2>&1
          sleep 15 & wait
          (( retries += 1 ))
        else
          break
        fi
        if [[ "${retries}" -gt 40 ]]; then
          echo "error: Another process is currently listening on the CNI socket, exiting" 2>&1
          exit 1
        fi
      done
      # if the node config doesn't exist yet, wait until it does
      retries=0
      while true; do
        if [[ ! -f /etc/origin/node/node-config.yaml ]]; then
          echo "warning: Cannot find existing node-config.yaml, waiting 15s ..." 2>&1
          sleep 15 & wait
          (( retries += 1 ))
        else
          break
        fi
        if [[ "${retries}" -gt 40 ]]; then
          echo "error: No existing node-config.yaml, exiting" 2>&1
          exit 1
        fi
      done

      # Take over network functions on the node
      rm -Rf /etc/cni/net.d/80-openshift-network.conf
      cp -Rf /opt/cni/bin/* /host/opt/cni/bin/

      if [[ -f /etc/sysconfig/origin-node ]]; then
        set -o allexport
        source /etc/sysconfig/origin-node
      fi

      exec openshift start network --config=/etc/origin/node/node-config.yaml --kubeconfig=/etc/origin/node/sdn.kubeconfig --loglevel=${DEBUG_LOGLEVEL:-2}
    env:
    - name: OPENSHIFT_DNS_DOMAIN
      value: cluster.local
    image: {{ .Config.Images.Node | quote }}
    name: sdn
    ports:
    - containerPort: 10256
      hostPort: 10256
      name: healthz
    resources:
      requests:
        cpu: 10m
        memory: 100Mi
    securityContext:
      privileged: true
      runAsUser: 0
    volumeMounts:
    - mountPath: /etc/origin/node/
      name: host-config
      readOnly: true
    - mountPath: /etc/sysconfig/origin-node
      name: host-sysconfig-node
      readOnly: true
    - mountPath: /var/run
      name: host-var-run
    - mountPath: /var/run/dbus/
      name: host-var-run-dbus
      readOnly: true
    - mountPath: /var/run/openvswitch/
      name: host-var-run-ovs
      readOnly: true
    - mountPath: /var/run/kubernetes/
      name: host-var-run-kubernetes
      readOnly: true
    - mountPath: /var/run/openshift-sdn
      name: host-var-run-openshift-sdn
    - mountPath: /host/opt/cni/bin
      name: host-opt-cni-bin
    - mountPath: /etc/cni/net.d
      name: host-etc-cni-netd
    - mountPath: /var/lib/cni/networks/openshift-sdn
      name: host-var-lib-cni-networks-openshift-sdn
  hostNetwork: true
  hostPID: true
  priorityClassName: system-node-critical
  volumes:
  - hostPath:
      path: /etc/origin/node
    name: host-config
  - hostPath:
      path: /etc/sysconfig/origin-node
    name: host-sysconfig-node
  - hostPath:
      path: /lib/modules
    name: host-modules
  - hostPath:
      path: /var/run
    name: host-var-run
  - hostPath:
      path: /var/run/dbus
    name: host-var-run-dbus
  - hostPath:
      path: /var/run/openvswitch
    name: host-var-run-ovs
  - hostPath:
      path: /var/run/kubernetes
    name: host-var-run-kubernetes
  - hostPath:
      path: /var/run/openshift-sdn
    name: host-var-run-openshift-sdn
  - hostPath:
      path: /opt/cni/bin
    name: host-opt-cni-bin
  - hostPath:
      path: /etc/cni/net.d
    name: host-etc-cni-netd
  - hostPath:
      path: /var/lib/cni/networks/openshift-sdn
    name: host-var-lib-cni-networks-openshift-sdn
EOF

# note: ${SERVICE_TYPE}-node crash loops until master is up
systemctl enable ${SERVICE_TYPE}-node.service
systemctl start ${SERVICE_TYPE}-node.service &

# disabling rsyslog since we manage everything through journald
systemctl disable rsyslog.service
systemctl stop rsyslog.service

#load the tuned profile which is recommended the host joins the cluster
{{- if eq .Extra.Role "infra" }}
tuned-adm profile openshift-control-plane
{{- else }}
tuned-adm profile openshift-node
{{- end }}
