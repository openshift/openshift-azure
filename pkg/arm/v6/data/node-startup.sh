#!/bin/bash -ex

{{ if gt (len .Config.SecurityPatchPackages) 0 }}
# enable RHUI extended update support (EUS) and lock current minor version
yum -y --config='https://rhelimage.blob.core.windows.net/repositories/rhui-microsoft-azure-rhel7-eus.config' install 'rhui-azure-rhel7-eus'
echo $(. /etc/os-release && echo $VERSION_ID) > /etc/yum/vars/releasever
yum update -y

# install cve-fixing rpm packages
{{ range .Config.SecurityPatchPackages }}
yum install -y {{.}}
{{end}}

# disable RHUI extended update support (EUS) and remove version lock
rm -rf /etc/yum/vars/releasever
yum -y --disablerepo='*' remove 'rhui-azure-rhel7-eus'
{{end}}

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
  "log-driver": "journald"
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

while ! docker pull {{ .Config.Images.Startup }}; do
  sleep 1
done
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

# note: atomic-openshift-node crash loops until master is up
systemctl enable atomic-openshift-node.service
systemctl start atomic-openshift-node.service &

# disabling rsyslog since we manage everything through journald
systemctl disable rsyslog.service
systemctl stop rsyslog.service
