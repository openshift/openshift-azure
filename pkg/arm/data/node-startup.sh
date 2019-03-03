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
base64 -d <<< {{ .Config.Images.ImagePullSecret | Base64Encode }} >${DPATH}/config.json
chmod 0600 ${DPATH}/config.json

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

echo 'nameserver 168.63.129.16' >/etc/origin/node/resolv.conf
mkdir -p /etc/origin/cloudprovider

cat >/etc/origin/cloudprovider/azure.conf <<'EOF'
{{ .Derived.WorkerCloudProviderConf .ContainerService | String }}
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
