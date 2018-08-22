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

# note: ${SERVICE_TYPE}-node crash loops until master is up
systemctl enable ${SERVICE_TYPE}-node.service
systemctl start ${SERVICE_TYPE}-node.service &
