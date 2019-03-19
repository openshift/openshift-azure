#!/bin/bash -ex

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
docker run --privileged --rm --network host -v /:/host:z -e SASURI {{ .Config.Images.Startup }}
unset SASURI

update-ca-trust

{{- if eq .Extra.Role "infra" }}
tuned-adm profile openshift-control-plane
{{- else }}
tuned-adm profile openshift-node
{{- end }}

# note: atomic-openshift-node crash loops until master is up
systemctl enable atomic-openshift-node.service
systemctl start atomic-openshift-node.service &

# disabling rsyslog since we manage everything through journald
systemctl disable rsyslog.service
systemctl stop rsyslog.service
