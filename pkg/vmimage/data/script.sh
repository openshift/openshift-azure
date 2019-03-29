#!/bin/bash -ex

exec 2>&1

export HOME=/root
cd

echo ,+ | sfdisk --force -u S -N 2 /dev/sda || true
partprobe
xfs_growfs /dev/sda2

yum -y update -x WALinuxAgent # updating WALinuxAgent kills this script
yum -y install git golang libguestfs-tools-c libvirt-daemon-config-network virt-install

rpm --import https://packages.microsoft.com/keys/microsoft.asc
cat >/etc/yum.repos.d/azure-cli.repo <<'EOF'
[azure-cli]
baseurl=https://packages.microsoft.com/yumrepos/azure-cli
enabled=1
gpgcheck=1
gpgkey=https://packages.microsoft.com/keys/microsoft.asc
EOF
yum -y install azure-cli

systemctl start libvirtd.service

mkdir data
base64 -d <<'EOF' | tar -C data -xz
{{ .Archive | Base64Encode }}
EOF

cat >client-cert.pem <<'EOF'
{{ .Builder.ClientCert | CertAsBytes | String }}
EOF

cat >client-key.pem <<'EOF'
{{ .Builder.ClientKey | PrivateKeyAsBytes | String }}
EOF

go get github.com/jim-minter/tlsproxy
go/bin/tlsproxy -insecure -key client-key.pem -cert client-cert.pem https://cdn.redhat.com/ &
while [[ "$(fuser -n tcp 8080)" == "" ]]; do
  sleep 1
done

firewall-cmd --zone=public --add-port=8080/tcp

IMAGE="{{ .Builder.Image }}"
DISKGIB=${DISKGIB:-32}
IP=$(ifconfig eth0 | awk '/inet / { print $2 }')

cat >rhel7.ks <<KICKSTART
bootloader
firstboot --disable
keyboard us
lang en_US.UTF-8
network --activate --device=eth0 --onboot=on
part / --fstype=xfs --size=10240
part /var --fstype=xfs --fsoptions=grpquota --grow
poweroff
rootpw --lock
text
timezone Etc/UTC
url --url=http://$IP:8080/content/dist/rhel/server/7/7.6/x86_64/kickstart
zerombr
%addon com_redhat_kdump --disable
%end
%packages --excludedocs --nocore
@^minimal
-NetworkManager-team
-Red_Hat_Enterprise_Linux-Release_Notes-7-en-US
-audit*
-biosdevname
-btrfs-progs
-dracut-config-rescue
-dracut-network
-iprutils
-jansson
-kbd*
-kernel-tools*
-kexec-tools
-libdaemon
-libnl3-cli
-libsysfs
-libteam
-libxslt
-lshw
-lsscsi
-lzo
-mariadb-libs
-microcode_ctl
-pciutils-libs
-postfix
-python-dateutil
-python-inotify
-python-lxml
-python-magic
-python-six
-redhat-support*
-sg3_utils*
-snappy
-teamd
-xdg-utils
-*-firmware
%end
%post
set -ex
exec </dev/console &>/dev/console
stty sane
#trap bash EXIT

echo 'add_drivers+="hv_storvsc hv_vmbus"' >/etc/dracut.conf.d/azure-drivers.conf
dracut --force

base64 -d <<'EOF' | cat >/etc/yum.repos.d/kickstart.repo
$(base64 <data/etc/yum.repos.d/kickstart.repo)
EOF

cat >/var/lib/yum/client-cert.pem <<'EOF'
$(cat client-cert.pem)
EOF

cat >/var/lib/yum/client-key.pem <<'EOF'
$(cat client-key.pem)
EOF

yum -y update
yum -y install \
    ansible \
    atomic \
    atomic-openshift-clients \
    atomic-openshift-docker-excluder \
    atomic-openshift-node \
    bind-utils \
    ceph-common \
    chrony \
    cifs-utils \
    conntrack-tools \
    device-mapper-multipath \
    dhclient \
    dnsmasq \
    docker \
    e2fsprogs \
    firewalld \
    glusterfs-fuse \
    grub2 \
    httpd-tools \
    insights-client \
    iptables-services \
    irqbalance \
    iscsi-initiator-utils \
    kernel \
    lsof \
    NetworkManager-config-server \
    NetworkManager \
    nfs-utils \
    ntp \
    openssh-clients \
    qemu-guest-agent \
    rootfiles \
    rsyslog \
    samba-client \
    strace \
    sudo \
    tcpdump \
    tuned \
    tree \
    find \
    WALinuxAgent \
    yum-utils
yum clean all

base64 -d <<'EOF' | tar -C / -x
$(tar -C data --owner=root --group=root -c . | base64 -w0)
EOF

# not commited with a : so that Windows users can check out the repo
mv /etc/docker/certs.d/docker-registry.default.svc-5000 /etc/docker/certs.d/docker-registry.default.svc:5000

rm /etc/yum.repos.d/kickstart.repo /var/lib/yum/client-cert.pem /var/lib/yum/client-key.pem

mkdir /var/lib/etcd

chmod 0755 /etc/NetworkManager/dispatcher.d/99-origin-dns.sh

setsebool -P \
    container_manage_cgroup=1 \
    virt_sandbox_use_fusefs=1 \
    virt_use_fusefs=1 \
    virt_use_samba=1

sed -i -e "s/^OPTIONS=.*/OPTIONS='--selinux-enabled --signature-verification=False'/" /etc/sysconfig/docker
sed -i -e "$ a \
ADD_REGISTRY='--add-registry registry.redhat.io'" /etc/sysconfig/docker

sed -i -e "s/^DOCKER_NETWORK_OPTIONS=.*/DOCKER_NETWORK_OPTIONS='--mtu=1450'/" /etc/sysconfig/docker-network

sed -i -e "s/^DOCKER_STORAGE_OPTIONS=.*/DOCKER_STORAGE_OPTIONS='--storage-driver overlay2'/" /etc/sysconfig/docker-storage

sed -i -e '/^HWADDR=/d' /etc/sysconfig/network-scripts/ifcfg-eth0

sed -i -e '/^#NAutoVTs=.*/ a \
NAutoVTs=0' /etc/systemd/logind.conf

sed -i -e 's/^ResourceDisk.Format=.*/ResourceDisk.Format=n/' /etc/waagent.conf

rpm -q kernel --last | sed -n '1 {s/^[^-]*-//; s/ .*$//; p}' >/var/tmp/kernel-version
rpm -q atomic-openshift-node --qf '%{VERSION}-%{RELEASE}.%{ARCH}' >/var/tmp/openshift-version

# HACK: We replace hyperkube binary with our custom binary to include backports we waiting to be ported to 3.11.
# Binary delta: https://github.com/mjudeikis/ose/commit/07a01264d5afc9b3e17e35c7777148eaf01ea2bf
# Related issues: 
# https://github.com/openshift/openshift-azure/issues/1362
# https://github.com/kubernetes/kubernetes/pull/70002

curl -o /bin/hyperkube https://hyperkubebin.blob.core.windows.net/hyperkube/hyperkube

# check if binary was not tampered with
if ! echo "ff1fd9512d22fe1f772c6971b38026a589400fa62f2b5b4842f87e0e84a60932 /bin/hyperkube" | sha256sum --check ; then
    echo "sha256sum does not match. Update node build script"
    exit 1
fi

if [ `cat /var/tmp/openshift-version` != "3.11.82-1.git.0.08bc31b.el7.x86_64" ]; then 
    echo "new rpm detected. Update node build script"
    exit 1
fi


>/var/tmp/kickstart_completed
%end
KICKSTART

python -c "import pty; pty.spawn([
    'virt-install',
    '--disk', '/var/lib/libvirt/images/$IMAGE.raw,size=$DISKGIB,format=raw',
    '--extra-args', 'console=ttyS0,115200n8 earlyprintk=ttyS0,115200 ks=file:/rhel7.ks',
    '--graphics', 'none',
    '--initrd-inject', 'rhel7.ks',
    '--location', 'http://$IP:8080/content/dist/rhel/server/7/7.6/x86_64/kickstart',
    '--memory', '1536',
    '--name', '$(date +%s)',
    '--os-variant', 'rhel7.6',
    '--transient',
])"

# fail if /var/tmp/kickstart_completed doesn't exist (i.e. if the kickstart crashed out)
virt-cat -a /var/lib/libvirt/images/$IMAGE.raw /var/tmp/kickstart_completed

KERNEL="$(virt-cat -a /var/lib/libvirt/images/$IMAGE.raw /var/tmp/kernel-version)"
OPENSHIFT="$(virt-cat -a /var/lib/libvirt/images/$IMAGE.raw /var/tmp/openshift-version)"

virt-sysprep -a /var/lib/libvirt/images/$IMAGE.raw
virt-sparsify --in-place /var/lib/libvirt/images/$IMAGE.raw

# RHEL `qemu-img convert` doesn't support converting to the Azure vhd subformat,
# so use `vhd-footer` utility do the same thing.

# qemu-img convert -f raw -O vpc -o subformat=fixed,force_size /var/lib/libvirt/images/$IMAGE.raw /var/lib/libvirt/images/$IMAGE.vhd

mv /var/lib/libvirt/images/$IMAGE.raw /var/lib/libvirt/images/$IMAGE.vhd

go get github.com/jim-minter/vhd-footer
go/bin/vhd-footer -size $((DISKGIB << 30)) >>/var/lib/libvirt/images/$IMAGE.vhd

set +x
az login --service-principal -u '{{ .ClientID }}' -p '{{ .ClientSecret }}' -t '{{ .TenantID }}'
set -x

az group create -g '{{ .Builder.ImageResourceGroup }}' -l '{{ .Builder.Location }}'
az storage account create -g '{{ .Builder.ImageResourceGroup }}' -n '{{ .Builder.ImageStorageAccount }}'
az storage container create --account-name '{{ .Builder.ImageStorageAccount }}' -n '{{ .Builder.ImageContainer }}'
set +x
KEY=$(az storage account keys list -g '{{ .Builder.ImageResourceGroup }}' -n '{{ .Builder.ImageStorageAccount }}' --query "[?keyName == 'key1'].value" -o tsv)
set -x

# `az storage blob upload` wastes time and bandwidth uploading all zero bytes of
# a large and mainly sparse disk image.  Use `azureblobupload` to speed things
# up.

go get github.com/jim-minter/azureblobupload
set +x
# az storage blob upload --account-name '{{ .Builder.ImageStorageAccount }}' --account-key $KEY --container-name '{{ .Builder.ImageContainer }}' --type page --file /var/lib/libvirt/images/$IMAGE.vhd
go/bin/azureblobupload -account-name '{{ .Builder.ImageStorageAccount }}' -account-key $KEY -container-name '{{ .Builder.ImageContainer }}' -file /var/lib/libvirt/images/$IMAGE.vhd -name $IMAGE.vhd
set -x

az image create -g '{{ .Builder.ImageResourceGroup }}' -n $IMAGE --source "https://{{ .Builder.ImageStorageAccount }}.blob.core.windows.net/{{ .Builder.ImageContainer }}/$IMAGE.vhd" --os-type Linux --tags "kernel=$KERNEL" "openshift=$OPENSHIFT" 'gitcommit={{ .Builder.GitCommit }}'
