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

{{- if $.Extra.TestConfig.RunningUnderTest }}
sed -i -e "s#DEBUG_LOGLEVEL=2#DEBUG_LOGLEVEL=4#" /etc/sysconfig/${SERVICE_TYPE}-node
{{- end }}

rm -rf /etc/etcd/* /etc/origin/master/*

base64 -d <<< {{ Base64Encode (YamlMarshal .Config.NodeBootstrapKubeconfig) }} >/etc/origin/node/bootstrap.kubeconfig
base64 -d <<< {{ Base64Encode (CertAsBytes .Config.Certificates.NodeBootstrap.Cert) }} >/etc/origin/node/node-bootstrapper.crt
base64 -d <<< {{ Base64Encode (PrivateKeyAsBytes .Config.Certificates.NodeBootstrap.Key) }} >/etc/origin/node/node-bootstrapper.key
chmod 0600 /etc/origin/node/node-bootstrapper.key /etc/origin/node/bootstrap.kubeconfig

base64 -d <<< {{ Base64Encode (CertAsBytes .Config.Certificates.Ca.Cert) }} >/etc/origin/node/ca.crt
cp /etc/origin/node/ca.crt /etc/pki/ca-trust/source/anchors/openshift-ca.crt
update-ca-trust

mkdir -p /root/.docker/
cat >/root/.docker/config.json <<EOF
{{ print (.Derived.RegistrySecret .ContainerService) }}
EOF

echo 'nameserver 168.63.129.16' >/etc/origin/node/resolv.conf
mkdir -p /etc/origin/cloudprovider

cat >/etc/origin/cloudprovider/azure.conf <<'EOF'
{{ .Derived.CloudProviderConf .ContainerService | String }}
EOF

{{- if not $.Extra.TestConfig.RunningUnderTest }}
mkdir -p /var/lib/logbridge
cat >/etc/origin/node/pods/logbridge.yaml <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: logbridge
  namespace: kube-system
spec:
  containers:
  - image: {{ .Config.Images.LogBridge | quote }}
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
{{- end }}

# note: ${SERVICE_TYPE}-node crash loops until master is up
systemctl enable ${SERVICE_TYPE}-node.service
systemctl start ${SERVICE_TYPE}-node.service &

# disabling rsyslog since we manage everything through journald
systemctl disable rsyslog.service
systemctl stop rsyslog.service
