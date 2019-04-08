#!/bin/bash -ex

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

docker pull {{ .Config.Images.Node }} &>/dev/null &

if ! grep /var/lib/etcd /etc/fstab; then
  mkfs.xfs /dev/disk/azure/scsi1/lun0 || true
  echo '/dev/disk/azure/scsi1/lun0  /var/lib/etcd  xfs  defaults  0 0' >>/etc/fstab
  mount /var/lib/etcd
  restorecon -R /var/lib/etcd
fi

for dst in tcp,8444; do
	proto=${dst%%,*}
	port=${dst##*,}
	iptables -A OS_FIREWALL_ALLOW -p $proto -m state --state NEW -m $proto --dport $port -j ACCEPT -w
done

iptables-save >/etc/sysconfig/iptables

# when starting node waagent and network utilities goes into race condition.
# if waagent runs before dns is known to the node we end up with empty string
while [[ $(hostname -d) == "" ]]; do sleep 1; done

while ! docker pull {{ .Config.Images.Startup }}; do
  sleep 1
done
set +x
export SASURI='{{ .Config.MasterStartupSASURI }}'
set -x
docker run --privileged --rm --network host -v /:/host:z -e SASURI --entrypoint startup {{ .Config.Images.Startup }}
unset SASURI

update-ca-trust

sed -i -re "s#( *server: ).*#\1https://$(hostname)#" /etc/origin/master/openshift-master.kubeconfig
sed -i -re "s#( *server: ).*#\1https://$(hostname)#" /etc/origin/node/node.kubeconfig

{{- if ne $.BackupBlobName "" }}
logger -t master-startup.sh "starting recovery on $(hostname)"

# step 1 get the backup
rm -Rf /var/lib/etcd/*
tempBackDir=/var/lib/etcd/backup
mkdir $tempBackDir

logger -t master-startup.sh "downloading backup"
while ! docker pull {{ .Config.Images.EtcdBackup }}; do
  sleep 1
done
docker run --rm --network host \
  -v /etc/origin/master:/etc/origin/master \
  -v /etc/origin/cloudprovider/:/_data/_out \
  -v $tempBackDir:/out:z \
  --entrypoint etcdbackup \
  {{ .Config.Images.EtcdBackup }} \
  -blobname={{ .BackupBlobName }} \
  -destination=/out/backup.db download
logger -t master-startup.sh "backup downloaded"

# step 2 restore
logger -t master-startup.sh "restoring snapshot"
while ! docker pull {{ .Config.Images.MasterEtcd }}; do
  sleep 1
done
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
rm -rf /var/lib/etcd/new $tempBackDir
restorecon -Rv /var/lib/etcd
logger -t master-startup.sh "restore done"
{{- end }}

tuned-adm profile openshift-control-plane

# note: atomic-openshift-node crash loops until master is up
systemctl enable atomic-openshift-node.service
systemctl start atomic-openshift-node.service &

# disabling rsyslog since we manage everything through journald
systemctl disable rsyslog.service
systemctl stop rsyslog.service
