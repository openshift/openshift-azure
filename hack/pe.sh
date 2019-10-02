#!/bin/bash -e

usage() {
    cat <<EOF >&2
usage:

$0 get-pe cluster-name

Examples:
$0 get-pe $RESOURCEGROUP

EOF
    exit 1
}

case "$1" in
get-pe)
    if [[ "$#" -ne 2 ]]; then
        usage
    fi
   
    MANAGEMENT_RG=$( az network private-link-service show -n mgmtpls -g $2 --query privateEndpointConnections[0].privateEndpoint.resourceGroup -o tsv)
    PE_NAME=$(az network private-link-service show -n mgmtpls -g $2 --query privateEndpointConnections[0].name -o tsv |  sed 's/\..*//')
    NIC_NAME=$(az network private-endpoint show -g $MANAGEMENT_RG -n $PE_NAME --query networkInterfaces[0].id -o tsv | rev | cut -d/ -f1 | rev)
    IP=$(az network nic show -g $MANAGEMENT_RG -n $NIC_NAME --query ipConfigurations[0].privateIpAddress -o tsv)
    echo ""
    echo "curl https://$IP -k"
    echo ""
    echo "You need VPN tunnel to $MANAGEMENT_RG"
       
    ;;
*)
    usage
    ;;

esac
