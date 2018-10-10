#!/bin/bash -e

usage() {
    cat <<EOF >&2
usage:

$0 get-config resourcegroup

EOF
    exit 1
}

if [[ -z "$AZURE_SUBSCRIPTION_ID" ]]; then
    AZURE_SUBSCRIPTION_ID=$(az account show --query id --output tsv)
fi

case "$1" in
get-config)
    if [[ "$#" -ne 2 ]]; then usage; fi

    OSACONFIG=$(mktemp -d)
    trap "rm -rf $OSACONFIG" EXIT

    export AZURE_STORAGE_ACCOUNT=$(az storage account list --subscription $AZURE_SUBSCRIPTION_ID -g $2 | jq '.[] | select(.tags.type == "config") | .name ' | tr -d '"')
    export AZURE_STORAGE_KEY=$(az storage account keys list --subscription $AZURE_SUBSCRIPTION_ID -n $AZURE_STORAGE_ACCOUNT -g $2 --query '[0].value' | tr -d '"')
    az storage blob download --subscription $AZURE_SUBSCRIPTION_ID --file $OSACONFIG/config -c config -n config --no-progress >/dev/null
    cat $OSACONFIG/config
    ;;

*)
    usage
    ;;

esac
