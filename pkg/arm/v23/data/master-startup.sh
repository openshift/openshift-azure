#!/bin/bash -ex

# For testing: log and drop all traffic that would route via the master ELB
# RESOURCEGROUP="your-resource-group"
# OS_FQDN="openshift.$RESOURCEGROUP.osadev.cloud"
#
# iptables -I INPUT -s $(dig +short "$OS_FQDN" | tail -1) -j LOG --log-prefix "KUBE FQDN DROP:"
# iptables -I INPUT -s $(dig +short "$OS_FQDN" | tail -1) -j DROP

{{ if .Config.SecurityPatchPackages }}
logger -t master-startup.sh "installing red hat cdn configuration on $(hostname)"
cat >/var/lib/yum/client-cert.pem <<'EOF'
{{ CertAsBytes .Config.Certificates.PackageRepository.Cert | String }}
EOF
cat >/var/lib/yum/client-key.pem <<'EOF'
{{ PrivateKeyAsBytes .Config.Certificates.PackageRepository.Key | String }}
EOF

logger -t master-startup.sh "installing ARO security updates [{{ StringsJoin .Config.SecurityPatchPackages ", " }}] on $(hostname)"
for attempt in {1..5}; do
  yum install -y -q {{ StringsJoin .Config.SecurityPatchPackages " " }} && break
  logger -t master-startup.sh "[attempt ${attempt}] ARO security updates installation failed"
  if [[ ${attempt} -lt 5 ]]; then sleep 1; else exit 1; fi
done

logger -t master-startup.sh "removing red hat cdn configuration on $(hostname)"
yum clean all
rm -rf /var/lib/yum/client-cert.pem /var/lib/yum/client-key.pem
{{end}}

# create container pull secret
mkdir -p /var/lib/origin/.docker
cat >/var/lib/origin/.docker/config.json <<'EOF'
{{ .Derived.CombinedImagePullSecret .Config | String }}
EOF
ln -s /var/lib/origin/.docker /root/

# TODO: delete the following group creation after it is baked into our VM images
groupadd -f docker

if ! grep /var/lib/docker /etc/fstab; then
  systemctl stop docker-cleanup.timer
  systemctl stop docker-cleanup.service
  systemctl stop docker.service
  mkfs.xfs -f /dev/disk/azure/resource-part1
  echo '/dev/disk/azure/resource-part1  /var/lib/docker  xfs  grpquota  0 0' >>/etc/fstab
  mount /var/lib/docker
  restorecon -R /var/lib/docker
  cat >/etc/docker/daemon.json <<'EOF'
{
  "log-driver": "journald",
  "disable-legacy-registry": true
}
EOF
  systemctl start docker.service
  systemctl start docker-cleanup.timer
fi

docker pull {{ .Config.Images.Node }} &>/dev/null &

if ! grep /var/lib/etcd /etc/fstab; then
  mkfs.xfs /dev/disk/azure/scsi1/lun0 || true
  echo '/dev/disk/azure/scsi1/lun0  /var/lib/etcd  xfs  defaults  0 0' >>/etc/fstab
  mount /var/lib/etcd
  restorecon -R /var/lib/etcd
fi

# accepting 8443 for the admission controller
for dst in tcp,8443 tcp,8444; do
	proto=${dst%%,*}
	port=${dst##*,}
	iptables -A OS_FIREWALL_ALLOW -p $proto -m state --state NEW -m $proto --dport $port -j ACCEPT -w
done

iptables-save >/etc/sysconfig/iptables

logger -t master-startup.sh "pulling {{ .Config.Images.Startup }}"
for attempt in {1..5}; do
  docker pull {{ .Config.Images.Startup }} && break
  logger -t master-startup.sh "[attempt ${attempt}] docker pull {{ .Config.Images.Startup }} failed"
  if [[ ${attempt} -lt 5 ]]; then sleep 60; else exit 1; fi
done

#
# NOTE: In future, move that information outside of environment variables
#
set +x
export SASURI='{{ .Config.MasterStartupSASURI }}'
set -x

# run the startup --init to bootstrap DNS setup
docker run --privileged --rm --network host -v /:/host -e SASURI {{ .Config.Images.Startup }} startup --init-network

# relable files to the right context
restorecon -R /etc /root

# restart network manager to pick up new host settings in the dhclient
/bin/systemctl restart NetworkManager
# set the /etc/resolv.conf
/etc/NetworkManager/dispatcher.d/99-origin-dns.sh
# restart dnsmasq to get the new settings in /etc/dnsmasq.conf
/bin/systemctl restart dnsmasq.service

# when starting node waagent and network utilities goes into race condition.
# if waagent runs before dns is known to the node we end up with empty string
while [[ $(hostname -d) == "" ]]; do sleep 1; done

docker run --privileged --rm --network host -v /:/host -e SASURI {{ .Config.Images.Startup }} startup

# relable files to the right context
restorecon -R /etc /root

unset SASURI

update-ca-trust

# setting up geneva logging stack as soon as the configuration is laid out by
# the startup container, the closer this is to the startup container, the more
# logs from the startup process we will ship to geneva

# these should run once a day only as per the docs
/usr/local/bin/azsecd config -s baseline -d P1D
/usr/local/bin/azsecd config -s software -d P1D
/usr/local/bin/azsecd config -s clamav -d P1D

# enable and start logging stack services
systemctl unmask mdsd.service azsecd.service azsecmond.service fluentd.service
systemctl enable mdsd.service azsecd.service azsecmond.service fluentd.service
systemctl start mdsd.service azsecd.service azsecmond.service fluentd.service

# Pin kubeconfigs to use local api-server
sed -i -re "s#( *server: ).*#\1https://$(hostname)#" /etc/origin/master/admin.kubeconfig
sed -i -re "s#( *server: ).*#\1https://$(hostname)#" /etc/origin/master/openshift-master.kubeconfig
sed -i -re "s#( *server: ).*#\1https://$(hostname)#" /etc/origin/node/node.kubeconfig
sed -i -re "s#( *server: ).*#\1https://$(hostname)#" /etc/origin/node/sdn.kubeconfig
sed -i -re "s#( *server: ).*#\1https://$(hostname)#" /root/.kube/config

{{- if ne $.BackupBlobName "" }}
logger -t master-startup.sh "starting recovery on $(hostname)"

# step 1 get the backup
rm -Rf /var/lib/etcd/*
tempBackDir=/var/lib/etcd/backup
mkdir $tempBackDir

logger -t master-startup.sh "pulling {{ .Config.Images.EtcdBackup }}"
for attempt in {1..5}; do
  docker pull {{ .Config.Images.EtcdBackup }} && break
  logger -t master-startup.sh "[attempt ${attempt}] docker pull {{ .Config.Images.EtcdBackup }} failed"
  if [[ ${attempt} -lt 5 ]]; then sleep 60; else exit 1; fi
done

logger -t master-startup.sh "downloading backup"
docker run --rm --network host \
  -v /etc/origin/master:/etc/origin/master \
  -v /etc/origin/cloudprovider/:/_data/_out \
  -v $tempBackDir:/out:z \
  {{ .Config.Images.EtcdBackup }} etcdbackup \
  --blobName={{ .BackupBlobName }} \
  --destination=/out/backup.db \
  --action=download
logger -t master-startup.sh "backup downloaded"

# step 2 restore
logger -t master-startup.sh "pulling {{ .Config.Images.MasterEtcd }}"
for attempt in {1..5}; do
  docker pull {{ .Config.Images.MasterEtcd }} && break
  logger -t master-startup.sh "[attempt ${attempt}] docker pull {{ .Config.Images.MasterEtcd }} failed"
  if [[ ${attempt} -lt 5 ]]; then sleep 60; else exit 1; fi
done

logger -t master-startup.sh "restoring snapshot"
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

# we also need openshift.local.volumes dir created before xfs quota code runs
mkdir -m 0750 -p /var/lib/origin/openshift.local.volumes

# disabling rsyslog since we manage everything through journald
systemctl disable rsyslog.service
systemctl stop rsyslog.service

# note: atomic-openshift-node crash loops until master is up
systemctl enable atomic-openshift-node.service
{{ if .Config.SecurityPatchPackages }}
logger -t master-startup.sh "scheduling $(hostname) reboot to complete ARO security updates"
shutdown --reboot +2
{{else}}
systemctl start atomic-openshift-node.service || true
{{end}}
