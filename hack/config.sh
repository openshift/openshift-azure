#!/bin/bash -e

usage() {
    cat <<EOF >&2
usage:
$0 get-config resourcegroup
EOF
    exit 1
}

case "$1" in
get-config)
    if [[ "$#" -ne 2 ]]; then usage; fi

    OSACONFIG=$(mktemp -d)
    trap "rm -rf $OSACONFIG" EXIT

    export AZURE_STORAGE_ACCOUNT=$(az storage account list -g $2 | jq '.[] | select(.tags.type == "config") | .name ' | tr -d '"')
    export AZURE_STORAGE_KEY=$(az storage account keys list -n $AZURE_STORAGE_ACCOUNT -g $2 --query '[0].value' | tr -d '"')
    az storage blob download --file $OSACONFIG/config -c config -n config --no-progress >/dev/null
    cat $OSACONFIG/config
    ;;

*)
    usage
    ;;

esac
