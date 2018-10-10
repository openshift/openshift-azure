#!/bin/bash -e

if [[ -z "$DNS_DOMAIN" ]]; then
    echo error: must set DNS_DOMAIN
    exit 1
fi

if [[ -z "$DNS_RESOURCEGROUP" ]]; then
    echo error: must set DNS_RESOURCEGROUP
    exit 1
fi

if [[ -z "$AZURE_SUBSCRIPTION_ID" ]]; then
    AZURE_SUBSCRIPTION_ID=$(az account show --query id --output tsv)
fi

usage() {
    cat <<EOF >&2
usage:

$0 zone-create zone
$0 zone-delete zone
$0 a-create zone name ip
$0 a-delete zone name
$0 cname-create zone name cname
$0 cname-delete zone name

example:

$0 zone-create testzone
$0 a-create testzone '*' 1.2.3.4
dig +short foo.testzone.$DNS_DOMAIN
$0 a-delete testzone '*'
$0 zone-delete testzone
$0 cname-create testzone '*' app.eastus.cloudapp.azure.com
$0 cname-delete testzone '*'

EOF
    exit 1
}

exec >/dev/null

case "$1" in
zone-create)
    if [[ "$#" -ne 2 ]]; then usage; fi

    # create the new zone
    az network dns zone create --subscription $AZURE_SUBSCRIPTION_ID -g "$DNS_RESOURCEGROUP" -n "$2.$DNS_DOMAIN"
    az network dns record-set soa update --subscription $AZURE_SUBSCRIPTION_ID -g "$DNS_RESOURCEGROUP" -z "$2.$DNS_DOMAIN" -f 60 -r 60 -x 60 -m 60
    az network dns record-set ns update --subscription $AZURE_SUBSCRIPTION_ID -g "$DNS_RESOURCEGROUP" -z "$2.$DNS_DOMAIN" -n @ --set ttl=60

    # register the new zone in the "$DNS_DOMAIN" zone
    NS=$(az network dns zone show --subscription $AZURE_SUBSCRIPTION_ID -g "$DNS_RESOURCEGROUP" -n "$2.$DNS_DOMAIN" --query 'nameServers[].{nsdname: @}')
    az network dns record-set ns create --subscription $AZURE_SUBSCRIPTION_ID -g "$DNS_RESOURCEGROUP" -z "$DNS_DOMAIN" -n "$2" --ttl 60
    az network dns record-set ns update --subscription $AZURE_SUBSCRIPTION_ID -g "$DNS_RESOURCEGROUP" -z "$DNS_DOMAIN" -n "$2" --set nsRecords="$NS"
    ;;

zone-delete)
    if [[ "$#" -ne 2 ]]; then usage; fi

    az network dns record-set ns delete --subscription $AZURE_SUBSCRIPTION_ID -g "$DNS_RESOURCEGROUP" -z "$DNS_DOMAIN" -n "$2" -y
    az network dns zone delete --subscription $AZURE_SUBSCRIPTION_ID -g "$DNS_RESOURCEGROUP" -n "$2.$DNS_DOMAIN" -y
    ;;

a-create)
    if [[ "$#" -ne 4 ]]; then usage; fi

    az network dns record-set a create --subscription $AZURE_SUBSCRIPTION_ID -g "$DNS_RESOURCEGROUP" -z "$2.$DNS_DOMAIN" -n "$3" --ttl 60
    az network dns record-set a update --subscription $AZURE_SUBSCRIPTION_ID -g "$DNS_RESOURCEGROUP" -z "$2.$DNS_DOMAIN" -n "$3" --set arecords='[{"ipv4Address": "'"$4"'"}]'
    ;;

a-delete)
    if [[ "$#" -ne 3 ]]; then usage; fi

    az network dns record-set a delete --subscription $AZURE_SUBSCRIPTION_ID -g "$DNS_RESOURCEGROUP" -z "$2.$DNS_DOMAIN" -n "$3" -y
    ;;

cname-create)
    if [[ "$#" -ne 4 ]]; then usage; fi

    az network dns record-set cname set-record --subscription $AZURE_SUBSCRIPTION_ID -g "$DNS_RESOURCEGROUP" -z "$2.$DNS_DOMAIN" -n "$3" -c "$4"
    ;;

cname-delete)
    if [[ "$#" -ne 3 ]]; then usage; fi

    az network dns record-set cname delete --subscription $AZURE_SUBSCRIPTION_ID -g "$DNS_RESOURCEGROUP" -z "$2.$DNS_DOMAIN" -n "$3"
    ;;

*)
    usage
    ;;

esac
