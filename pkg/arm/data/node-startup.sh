#!/bin/bash -ex

# TODO: /etc/dnsmasq.d/origin-upstream-dns.conf is currently hardcoded; it
# probably shouldn't be

SERVICE_TYPE=origin
if [ -f "/etc/sysconfig/atomic-openshift-node" ]; then
    SERVICE_TYPE=atomic-openshift
fi

if ! grep /var/lib/docker /etc/fstab; then
  mkfs.xfs -f /dev/disk/azure/resource-part1
  echo '/dev/disk/azure/resource-part1  /var/lib/docker  xfs  grpquota  0 0' >>/etc/fstab
  systemctl stop docker.service
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

echo 'BOOTSTRAP_CONFIG_NAME=node-config-{{ .Extra.Role }}' >>/etc/sysconfig/${SERVICE_TYPE}-node

sed -i -e "s#DEBUG_LOGLEVEL=2#DEBUG_LOGLEVEL=4#" /etc/sysconfig/${SERVICE_TYPE}-node

rm -rf /etc/etcd/* /etc/origin/master/*

base64 -d <<< {{ Base64Encode (YamlMarshal .Config.NodeBootstrapKubeconfig) }} >/etc/origin/node/bootstrap.kubeconfig
base64 -d <<< {{ Base64Encode (CertAsBytes .Config.Certificates.NodeBootstrap.Cert) }} >/etc/origin/node/node-bootstrapper.crt
base64 -d <<< {{ Base64Encode (PrivateKeyAsBytes .Config.Certificates.NodeBootstrap.Key) }} >/etc/origin/node/node-bootstrapper.key
chmod 0600 /etc/origin/node/node-bootstrapper.key /etc/origin/node/bootstrap.kubeconfig

base64 -d <<< {{ Base64Encode (CertAsBytes .Config.Certificates.Ca.Cert) }} >/etc/origin/node/ca.crt
cp /etc/origin/node/ca.crt /etc/pki/ca-trust/source/anchors/openshift-ca.crt
update-ca-trust

echo 'nameserver 168.63.129.16' >/etc/origin/node/resolv.conf
mkdir -p /etc/origin/cloudprovider

cat >/etc/origin/cloudprovider/azure.conf <<'EOF'
{{ .Config.CloudProviderConf | String }}
EOF

cat >/etc/origin/node/pods/ovs.yaml <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  annotations:
    kubernetes.io/description: |
      This static pod launches the openvswitch daemon.
  labels:
    app: ovs
    component: network
    openshift.io/component: network
    type: infra
  name: ovs
  namespace: openshift-sdn
spec:
    hostNetwork: true
    hostPID: true
    serviceAccountName: sdn
    containers:
      - name: openvswitch
        command:
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
          /usr/share/openvswitch/scripts/ovs-ctl start --system-id=random

          # Restrict the number of pthreads ovs-vswitchd creates to reduce the
          # amount of RSS it uses on hosts with many cores
          # https://bugzilla.redhat.com/show_bug.cgi?id=1571379
          # https://bugzilla.redhat.com/show_bug.cgi?id=1572797
          if [[ `nproc` -gt 12 ]]; then
              ovs-vsctl set Open_vSwitch . other_config:n-revalidator-threads=4
              ovs-vsctl set Open_vSwitch . other_config:n-handler-threads=10
          fi
          while true; do sleep 5; done
        image: {{ .Config.NodeImage | quote }}
        resources:
         limits:
           cpu: 200m
           memory: 400Mi
         requests:
           cpu: 100m
           memory: 300Mi
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
    volumes:
    - name: host-modules
      hostPath:
        path: /lib/modules
    - name: host-run-ovs
      hostPath:
        path: /run/openvswitch
    - name: host-sys
      hostPath:
        path: /sys
    - name: host-config-openvswitch
      hostPath:
        path: /etc/origin/openvswitch
EOF

cat >/etc/origin/node/pods/sdn.yaml <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  annotations:
    kubernetes.io/description: |
      This static pod launches the OpenShift networking components (kube-proxy, DNS, and openshift-sdn).
      It expects that OVS is running on the node.
    scheduler.alpha.kubernetes.io/critical-pod: ""
  labels:
    app: sdn
    component: network
    openshift.io/component: network
    type: infra
  name: sdn
  namespace: openshift-sdn
spec:
  hostNetwork: true
  hostPID: true
  serviceAccountName: sdn
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
        file=/etc/sysconfig/origin-node
        if [[ -f /etc/sysconfig/atomic-openshift-node ]]; then
          file=/etc/sysconfig/atomic-openshift-node
        elif [[ -f /etc/sysconfig/origin-node ]]; then
          file=/etc/sysconfig/origin-node
        else
          echo "info: Waiting for the node sysconfig file to be created" 2>&1
          sleep 15 & wait
          continue
        fi
        config_file="$(sed -nE 's|^CONFIG_FILE=([^#].+)|\1|p' "${file}" | head -1)"
        if [[ -z "${config_file}" ]]; then
          echo "info: Waiting for CONFIG_FILE to be set" 2>&1
          sleep 15 & wait
          continue
        fi
        if [[ ! -f ${config_file} ]]; then
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
      rm -Rf /etc/cni/net.d/*
      rm -Rf /host/opt/cni/bin/*
      cp -Rf /opt/cni/bin/* /host/opt/cni/bin/

      if [[ -f /etc/sysconfig/origin-node ]]; then
        set -o allexport
        source /etc/sysconfig/origin-node
      fi

      # use either the bootstrapped node kubeconfig or the static configuration
      file=/etc/origin/node/node.kubeconfig
      if [[ ! -f "${file}" ]]; then
        # use the static node config if it exists
        # TODO: remove when static node configuration is no longer supported
        for f in /etc/origin/node/system*.kubeconfig; do
          echo "info: Using ${f} for node configuration" 1>&2
          file="${f}"
          break
        done
      fi
      # Use the same config as the node, but with the service account token
      oc config "--config=${file}" view --flatten > /tmp/kubeconfig
      oc config --config=/tmp/kubeconfig set-credentials sa "--token=$( cat /var/run/secrets/kubernetes.io/serviceaccount/token )"
      oc config --config=/tmp/kubeconfig set-context "$( oc config --config=/tmp/kubeconfig current-context )" --user=sa
      # Launch the network process
      exec openshift start network --config=${config_file} --kubeconfig=/tmp/kubeconfig --loglevel=${DEBUG_LOGLEVEL:-2}
    env:
    - name: OPENSHIFT_DNS_DOMAIN
      value: cluster.local
    image: {{ .Config.NodeImage | quote }}
    name: sdn
    ports:
    - containerPort: 10256
      hostPort: 10256
      name: healthz
    resources:
      requests:
        cpu: 100m
        memory: 200Mi
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
  volumes:
  - name: host-config
    hostPath:
      path: /etc/origin/node
  - name: host-sysconfig-node
    hostPath:
      path: /etc/sysconfig/origin-node
  - name: host-modules
    hostPath:
      path: /lib/modules
  - name: host-var-run
    hostPath:
      path: /var/run
  - name: host-var-run-dbus
    hostPath:
      path: /var/run/dbus
  - name: host-var-run-ovs
    hostPath:
      path: /var/run/openvswitch
  - name: host-var-run-kubernetes
    hostPath:
      path: /var/run/kubernetes
  - name: host-var-run-openshift-sdn
    hostPath:
      path: /var/run/openshift-sdn
  - name: host-opt-cni-bin
    hostPath:
      path: /opt/cni/bin
  - name: host-etc-cni-netd
    hostPath:
      path: /etc/cni/net.d
  - name: host-var-lib-cni-networks-openshift-sdn
    hostPath:
      path: /var/lib/cni/networks/openshift-sdn
EOF

mkdir -p /var/lib/logbridge
cat >/etc/origin/node/pods/logbridge.yaml <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: logbridge
  namespace: kube-system
spec:
  containers:
  - image: {{ .Config.LogBridgeImage | quote }}
    imagePullPolicy: Always
    name: logbridge
    securityContext:
      privileged: true
    volumeMounts:
    - mountPath: /state
      name: state
    - mountPath: /cloudprovider
      name: master-cloud-provider
      readOnly: true
    - mountPath: /etc
      name: etc
      readOnly: true
    - mountPath: /var/log
      name: var-log
      readOnly: true
  hostNetwork: true
  volumes:
  - hostPath:
      path: /var/lib/logbridge
    name: state
  - hostPath:
      path: /etc/origin/cloudprovider
    name: master-cloud-provider
  - hostPath:
      path: /etc
    name: etc
  - hostPath:
      path: /var/log
    name: var-log
EOF

# note: ${SERVICE_TYPE}-node crash loops until master is up
systemctl enable ${SERVICE_TYPE}-node.service
systemctl start ${SERVICE_TYPE}-node.service &
