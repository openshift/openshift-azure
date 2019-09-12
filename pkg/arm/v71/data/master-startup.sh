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
# TODO: delete the following section after all clusters are run with a vm image with kickstart.repo baked in
cat > /etc/yum.repos.d/kickstart.repo <<'EOF'
[rhel-7-server-rpms]
name=Red Hat Enterprise Linux 7 Server (RPMs)
baseurl=https://cdn.redhat.com/content/dist/rhel/server/7/7Server/$basearch/os
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-redhat-release
sslcacert=/etc/rhsm/ca/redhat-uep.pem
sslclientcert=/var/lib/yum/client-cert.pem
sslclientkey=/var/lib/yum/client-key.pem
enabled=yes

[rhel-7-server-extras-rpms]
name=Red Hat Enterprise Linux 7 Server - Extras (RPMs)
baseurl=https://cdn.redhat.com/content/dist/rhel/server/7/7Server/$basearch/extras/os
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-redhat-release
sslcacert=/etc/rhsm/ca/redhat-uep.pem
sslclientcert=/var/lib/yum/client-cert.pem
sslclientkey=/var/lib/yum/client-key.pem
enabled=yes

[rhel-7-server-ose-3.11-rpms]
name=Red Hat OpenShift Container Platform 3.11 (RPMs)
baseurl=https://cdn.redhat.com/content/dist/rhel/server/7/7Server/$basearch/ose/3.11/os
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-redhat-release
sslcacert=/etc/rhsm/ca/redhat-uep.pem
sslclientcert=/var/lib/yum/client-cert.pem
sslclientkey=/var/lib/yum/client-key.pem
enabled=yes

[azurecore]
name=azurecore
baseurl=https://packages.microsoft.com/yumrepos/azurecore
enabled=yes
gpgcheck=no
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
docker run --privileged --rm --network host -v /:/host:z -e SASURI {{ .Config.Images.Startup }} startup
unset SASURI

update-ca-trust

# Pin kubeconfigs to use local api-server
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

logger -t master-startup.sh "downloading backup"
while ! docker pull {{ .Config.Images.EtcdBackup }}; do
  sleep 1
done
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

# we also need openshift.local.volumes dir created before xfs quota code runs
mkdir -m 0750 -p /var/lib/origin/openshift.local.volumes

# disabling rsyslog since we manage everything through journald
systemctl disable rsyslog.service
systemctl stop rsyslog.service

# setting up geneva logging stack
systemctl unmask mdsd.service azsecd.service azsecmond.service fluentd.service
systemctl enable mdsd.service azsecd.service azsecmond.service fluentd.service
systemctl start mdsd.service azsecd.service azsecmond.service fluentd.service


# note: atomic-openshift-node crash loops until master is up
systemctl enable atomic-openshift-node.service
systemctl start atomic-openshift-node.service &
{{ if .Config.SecurityPatchPackages }}
needs-restarting --reboothint &>/dev/null || {
  logger -t master-startup.sh "rebooting $(hostname) to complete ARO security updates"
  shutdown --reboot now
}
{{end}}

