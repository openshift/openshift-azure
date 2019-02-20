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
  cat >/etc/docker/daemon.json <<'EOF'
{
  "log-driver": "journald"
}
EOF
  systemctl start docker.service
fi

# file should be /root/.docker/config.json, but actually need it in
# /var/lib/origin thanks to https://github.com/kubernetes/kubernetes/issues/45487
DPATH=/var/lib/origin/.docker
mkdir -p $DPATH
base64 -d <<< {{ .Config.Images.ImagePullSecret | Base64Encode }} >${DPATH}/config.json
chmod 0600 ${DPATH}/config.json

# TODO: consider fact that /dev/disk/azure/scsi1/lun0 is currently hardcoded;
# partition /dev/disk/azure/scsi1/lun0; consider future strategy for resizes if
# needed
if ! grep /var/lib/etcd /etc/fstab; then
  mkfs.xfs /dev/disk/azure/scsi1/lun0 || true
  echo '/dev/disk/azure/scsi1/lun0  /var/lib/etcd  xfs  defaults  0 0' >>/etc/fstab
  mount /var/lib/etcd
  restorecon -R /var/lib/etcd
fi

echo "BOOTSTRAP_CONFIG_NAME=node-config-master" >>/etc/sysconfig/${SERVICE_TYPE}-node

{{- if $.Extra.TestConfig.RunningUnderTest }}
sed -i -e "s#DEBUG_LOGLEVEL=2#DEBUG_LOGLEVEL=4#" /etc/sysconfig/${SERVICE_TYPE}-node
{{- end }}

for dst in tcp,8444; do
	proto=${dst%%,*}
	port=${dst##*,}
	iptables -A OS_FIREWALL_ALLOW -p $proto -m state --state NEW -m $proto --dport $port -j ACCEPT
done

iptables-save >/etc/sysconfig/iptables

rm -rf /etc/etcd/* /etc/origin/master/*

mkdir -p /etc/origin/cloudprovider
cat >/etc/origin/cloudprovider/azure.conf <<'EOF'
{{ .Derived.MasterCloudProviderConf .ContainerService | String }}
EOF

# when starting node waagent and network utilities goes into race condition.
# if waagent runs before dns is known to the node we end up with empty string
while [[ $(hostname -d) == "" ]]; do sleep 1; done

logger -t master-startup.sh "running startup container"
while ! docker pull {{ .Config.Images.Startup }}; do
  logger -t master-startup.sh "waiting for image pull to work"
  sleep 10
done
docker run --privileged --rm --network host -v /etc:/etc:z {{ .Config.Images.Startup }}
logger -t master-startup.sh "finished running startup container rc:$?"

update-ca-trust
echo 'nameserver 168.63.129.16' >/etc/origin/node/resolv.conf

sed -i -re "s#( *server: ).*#\1https://$(hostname)#" /etc/origin/master/openshift-master.kubeconfig
sed -i -re "s#( *server: ).*#\1https://$(hostname)#" /etc/origin/node/node.kubeconfig
# HACK: copy node.kubeconfig to bootstrap.kubeconfig so that openshift start node used in the sync
# daemonset will not fail and set the master node labels correctly.
cp /etc/origin/node/node.kubeconfig /etc/origin/node/bootstrap.kubeconfig

{{- if $.Extra.IsRecovery }}
logger -t master-startup.sh "starting recovery on $(hostname)"
# step 1 get the backup
rm -Rf /var/lib/etcd/*
tempBackDir=/var/lib/etcd/backup
mkdir $tempBackDir
while ! docker pull {{ .Config.Images.EtcdBackup }}; do sleep 10; done
docker run --rm --network host \
  -v /etc/origin/master:/etc/origin/master \
  -v /etc/origin/cloudprovider/:/_data/_out \
  -v $tempBackDir:/out:z \
  {{ .Config.Images.EtcdBackup }} \
  -blobname={{ .Extra.BackupBlobName }} \
  -destination=/out/backup.db "download"
logger -t master-startup.sh "backup downloaded to " $(ls $tempBackDir)

# step 2 restore
logger -t master-startup.sh "restoring snapshot"
while ! docker pull {{ .Config.Images.MasterEtcd }}; do sleep 10; done
docker run --rm --network host \
  -v /etc/etcd:/etc/etcd \
  -v /var/lib/etcd:/var/lib/etcd:z \
  -e ETCDCTL_API="3" \
  {{ .Config.Images.MasterEtcd }} \
  etcdctl snapshot restore $tempBackDir/backup.db \
  --data-dir /var/lib/etcd/new \
  --name $(hostname) \
  --initial-cluster "master-000000=https://master-000000:2380,master-000001=https://master-000001:2380,master-000002=https://master-000002:2380" \
  --initial-cluster-token etcd-for-azure \
  --initial-advertise-peer-urls https://$(hostname):2380

mv /var/lib/etcd/new/* /var/lib/etcd/
rm -rf /var/lib/etcd/new
rm -rf $tempBackDir
restorecon -Rv /var/lib/etcd
logger -t master-startup.sh "restore done"
{{- end }}

#set the recommended openshift-control-plane profile on first boot
tuned-adm profile openshift-control-plane

# note: ${SERVICE_TYPE}-node crash loops until master is up
systemctl enable ${SERVICE_TYPE}-node.service
systemctl start ${SERVICE_TYPE}-node.service &

# disabling rsyslog since we manage everything through journald
systemctl disable rsyslog.service
systemctl stop rsyslog.service

mkdir -p /root/.kube
cp /etc/origin/master/admin.kubeconfig /root/.kube/config

