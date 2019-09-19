#!/bin/bash -ex

# drop all traffic to the kubelet read-only port which doesn't originate from localhost
# TODO: delete these iptables rules after the azure-monitor team fixes their agent to access the authenticated port
iptables --insert INPUT --protocol tcp --dport 10255 --jump DROP
iptables --insert INPUT --protocol tcp --source localhost --dport 10255 --jump ACCEPT

{{ if .Config.SecurityPatchPackages }}
logger -t node-startup.sh "installing red hat cdn configuration on $(hostname)"
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

logger -t node-startup.sh "installing ARO security updates [{{ StringsJoin .Config.SecurityPatchPackages ", " }}] on $(hostname)"
for attempt in {1..5}; do
  yum install -y -q {{ StringsJoin .Config.SecurityPatchPackages " " }} && break
  logger -t node-startup.sh "[attempt ${attempt}] ARO security updates installation failed"
  if [[ ${attempt} -lt 5 ]]; then sleep 1; else exit 1; fi
done

logger -t node-startup.sh "removing red hat cdn configuration on $(hostname)"
yum clean all
rm -rf /var/lib/yum/client-cert.pem /var/lib/yum/client-key.pem
{{end}}

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
{{- if eq .Role "infra" }}
  cat >/etc/docker/daemon.json <<'EOF'
{
  "log-driver": "journald",
  "disable-legacy-registry": true
}
EOF
{{- end }}
  systemctl start docker.service
  systemctl start docker-cleanup.timer
fi

docker pull {{ .Config.Images.Node }} &>/dev/null &

# when starting node waagent and network utilities goes into race condition.
# if waagent runs before dns is known to the node we end up with empty string
while [[ $(hostname -d) == "" ]]; do sleep 1; done

logger -t node-startup.sh "pulling {{ .Config.Images.Startup }}"
for attempt in {1..5}; do
  docker pull {{ .Config.Images.Startup }} && break
  logger -t node-startup.sh "[attempt ${attempt}] docker pull {{ .Config.Images.Startup }} failed"
  if [[ ${attempt} -lt 5 ]]; then sleep 60; else exit 1; fi
done

#
# NOTE: In future, move that information outside of environment variables
#
set +x
export SASURI='{{ .Config.WorkerStartupSASURI }}'
set -x
docker run --privileged --rm --network host -v /:/host:z -e SASURI {{ .Config.Images.Startup }} startup
unset SASURI

update-ca-trust

{{- if eq .Role "infra" }}
tuned-adm profile openshift-control-plane
{{- else }}
tuned-adm profile openshift-node
{{- end }}

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
{{ if .Config.SecurityPatchPackages }}
logger -t node-startup.sh "scheduling $(hostname) reboot to complete ARO security updates"
shutdown --reboot +2
{{else}}
systemctl start atomic-openshift-node.service || true
{{end}}

