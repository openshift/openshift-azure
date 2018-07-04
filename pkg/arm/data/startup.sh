#!/bin/bash -ex

# TODO: /etc/dnsmasq.d/origin-upstream-dns.conf is currently hardcoded; it
# probably shouldn't be

SERVICE_TYPE=origin
if [ -f "/etc/sysconfig/atomic-openshift-node" ]; then
    SERVICE_TYPE=atomic-openshift
fi

systemctl stop docker.service
umount /mnt/resource || true
mkfs.xfs -f /dev/sdb1
echo '/dev/sdb1  /var/lib/docker  xfs  grpquota  0 0' >>/etc/fstab
mount /var/lib/docker
restorecon -R /var/lib/docker
systemctl start docker.service

{{if eq .Extra.Role "infra"}}
echo "BOOTSTRAP_CONFIG_NAME=node-config-infra" >>/etc/sysconfig/${SERVICE_TYPE}-node
{{else}}
echo "BOOTSTRAP_CONFIG_NAME=node-config-compute" >>/etc/sysconfig/${SERVICE_TYPE}-node
{{end}}

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

# TODO: this is duplicated in the helm charts for the master pods, and that's not ideal
cat >/etc/origin/cloudprovider/azure.conf <<'EOF'
tenantId: {{ .Manifest.TenantID }}
subscriptionId: {{ .Manifest.SubscriptionID }}
aadClientId: {{ .Manifest.ClientID }}
aadClientSecret: {{ .Manifest.ClientSecret }}
aadTenantId: {{ .Manifest.TenantID }}
resourceGroup: {{ .Manifest.ResourceGroup }}
location: {{ .Manifest.Location }}
securityGroupName: nsg-compute
primaryScaleSetName: ss-compute                                                                                                       
vmType: vmss
EOF

# note: ${SERVICE_TYPE}-node crash loops until master is up
systemctl enable ${SERVICE_TYPE}-node.service
systemctl start ${SERVICE_TYPE}-node.service &
