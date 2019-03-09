#!/bin/bash -e

usage() {
    cat <<EOF >&2
usage:
$0 image

example:
$0 rhel7-3.11-201901010000

EOF
    exit 1
}

if [[ "$#" -ne 1 ]]; then
    usage
fi

echo "Disk version: $(az image show -g images -n "$1" --query tags.openshift -o tsv | sed -e 's/-.*//; s/\.//').${1: -12:8}"
echo "OS VHD URL:   $(az storage blob url --account-name openshiftimages --container-name images --name "$1.vhd" -o tsv)?$(az storage container generate-sas --account-name openshiftimages --name images --start $(date -ud '-1 day' '+%Y-%m-%dT%H:%MZ') --expiry $(date -ud '+15 day' '+%Y-%m-%dT%H:%MZ') --permissions rl -o tsv)"
