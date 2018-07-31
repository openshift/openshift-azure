#!/bin/bash -ex

# TODO: /etc/dnsmasq.d/origin-upstream-dns.conf is currently hardcoded; it
# probably shouldn't be

SERVICE_TYPE=origin
if [ -f "/etc/sysconfig/atomic-openshift-node" ]; then
    SERVICE_TYPE=atomic-openshift
fi

umount /mnt/resource || true
mkfs.xfs -f /dev/disk/azure/resource-part1
echo '/dev/disk/azure/resource-part1  /var/lib/docker  xfs  grpquota  0 0' >>/etc/fstab
systemctl stop docker.service
mount /var/lib/docker
restorecon -R /var/lib/docker
systemctl start docker.service

echo 'BOOTSTRAP_CONFIG_NAME=node-config-{{ .Extra.Role }}' >>/etc/sysconfig/${SERVICE_TYPE}-node

sed -i -e "s#DEBUG_LOGLEVEL=2#DEBUG_LOGLEVEL=4#" /etc/sysconfig/${SERVICE_TYPE}-node

rm -rf /etc/etcd/* /etc/origin/master/*

base64 -d <<< {{ Base64Encode (YamlMarshal .Config.NodeBootstrapKubeconfig) }} >/etc/origin/node/bootstrap.kubeconfig
base64 -d <<< {{ Base64Encode (CertAsBytes .Config.NodeBootstrapCert) }} >/etc/origin/node/node-bootstrapper.crt
base64 -d <<< {{ Base64Encode (PrivateKeyAsBytes .Config.NodeBootstrapKey) }} >/etc/origin/node/node-bootstrapper.key
chmod 0600 /etc/origin/node/node-bootstrapper.key /etc/origin/node/bootstrap.kubeconfig



base64 -d <<< {{ Base64Encode (CertAsBytes .Config.CaCert) }} >/etc/origin/node/ca.crt
cp /etc/origin/node/ca.crt /etc/pki/ca-trust/source/anchors/openshift-ca.crt
update-ca-trust

echo 'nameserver 168.63.129.16' >/etc/origin/node/resolv.conf
mkdir -p /etc/origin/cloudprovider

# TODO: this is duplicated, and that's not ideal
cat >/etc/origin/cloudprovider/azure.conf <<'EOF'
tenantId: {{ .Config.TenantID | quote }}
subscriptionId: {{ .Config.SubscriptionID | quote }}
aadClientId: {{ .ContainerService.Properties.ServicePrincipalProfile.ClientID | quote }}
aadClientSecret: {{ .ContainerService.Properties.ServicePrincipalProfile.Secret | quote }}
aadTenantId: {{ .Config.TenantID | quote }}
resourceGroup: {{ .Config.ResourceGroup | quote }}
location: {{ .ContainerService.Location | quote }}
securityGroupName: nsg-compute
primaryScaleSetName: ss-compute
vmType: vmss
EOF

# note: ${SERVICE_TYPE}-node crash loops until master is up
systemctl enable ${SERVICE_TYPE}-node.service
systemctl start ${SERVICE_TYPE}-node.service &
