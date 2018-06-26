#!/bin/bash -e

RG=dns
ROOT=osadev.cloud

usage() {
    cat <<EOF >&2
usage:

$0 zone-create zone
$0 zone-delete zone
$0 a-create zone name ip
$0 a-delete zone name

example:

$0 zone-create testzone
$0 a-create testzone '*' 1.2.3.4
dig +short foo.testzone.$ROOT
$0 a-delete testzone '*'
$0 zone-delete testzone

EOF
    exit 1
}

exec >/dev/null

case "$1" in
zone-create)
    if [[ "$#" -ne 2 ]]; then usage; fi

    # create the new zone
    az network dns zone create -g "$RG" -n "$2.$ROOT"
    az network dns record-set soa update -g "$RG" -z "$2.$ROOT" -f 60 -r 60 -x 60 -m 60
    az network dns record-set ns update -g "$RG" -z "$2.$ROOT" -n @ --set ttl=60

    # register the new zone in the "$ROOT" zone
    NS=$(az network dns zone show -g "$RG" -n "$2.$ROOT" --query 'nameServers[].{nsdname: @}')
    az network dns record-set ns create -g "$RG" -z "$ROOT" -n "$2" --ttl 60
    az network dns record-set ns update -g "$RG" -z "$ROOT" -n "$2" --set nsRecords="$NS"
    ;;

zone-delete)
    if [[ "$#" -ne 2 ]]; then usage; fi

    az network dns record-set ns delete -g "$RG" -z "$ROOT" -n "$2" -y
    az network dns zone delete -g "$RG" -n "$2.$ROOT" -y
    ;;

a-create)
    if [[ "$#" -ne 4 ]]; then usage; fi

    az network dns record-set a create -g "$RG" -z "$2.$ROOT" -n "$3" --ttl 60
    az network dns record-set a update -g "$RG" -z "$2.$ROOT" -n "$3" --set arecords='[{"ipv4Address": "'"$4"'"}]'
    ;;

a-delete)
    if [[ "$#" -ne 3 ]]; then usage; fi

    az network dns record-set a delete -g "$RG" -z "$2.$ROOT" -n "$3" -y
    ;;

*)
    usage
    ;;

esac
