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

yum check-update > /tmp/check || true
yum updateinfo > /tmp/info || true


# install openscap and run
yum install -y openscap-scanner openscap-utils scap-security-guide

oscap xccdf eval \
  --profile stig-rhel7-disa \
  --results /tmp/scap-results.xml \
  --report /tmp/scap-report.html \
  --oval-results --fetch-remote-resources \
  --cpe /usr/share/xml/scap/ssg/content/ssg-rhel7-cpe-dictionary.xml \
  /usr/share/xml/scap/ssg/content/ssg-rhel7-xccdf.xml 2>&1 > /tmp/oscap.log || true
