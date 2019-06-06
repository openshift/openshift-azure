#!/bin/bash -ex

exec 2>&1

export HOME=/root
cd

mkdir data
base64 -d <<'EOF' | tar -C data -xz
{{ .Archive | Base64Encode }}
EOF

cat >/var/lib/yum/client-cert.pem <<'EOF'
{{ .Builder.ClientCert | CertAsBytes | String }}
EOF

cat >/var/lib/yum/client-key.pem <<'EOF'
{{ .Builder.ClientKey | PrivateKeyAsBytes | String }}
EOF

cp data/etc/yum.repos.d/kickstart.repo /etc/yum.repos.d/kickstart.repo

yum check-update > /tmp/yum_check_update || true
yum updateinfo > /tmp/yum_update_info || true


# install openscap and run
yum install -y openscap-scanner openscap-utils scap-security-guide

oscap xccdf eval \
  --profile xccdf_cloud.osadev_profile_stig_customized_aro \
  --results /tmp/scap-results.xml \
  --report /tmp/scap-report.html \
  --tailoring-file /root/data/ssg-rhel7-ds-aro.xml \
  --oval-results --fetch-remote-resources \
  --cpe /usr/share/xml/scap/ssg/content/ssg-rhel7-cpe-dictionary.xml \
  /usr/share/xml/scap/ssg/content/ssg-rhel7-ds.xml > /tmp/oscap.log || true
  
