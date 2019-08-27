#!/bin/bash -e

usage() {
    cat <<EOF >&2
usage:

$0 [start,status,stop] region

Examples:
$0 start region
$0 stop region
$0 status region
$0 get-pe $RESOURCEGROUP

EOF
    exit 1
}

case "$1" in
start)
    if [[ "$#" -ne 2 ]]; then
        usage
    fi
    # cache sudo password
    if [[ ! $(sudo echo 0) ]]; then exit; fi
    sudo openvpn --log /var/log/openvpn.log --config secrets/vpn-$2.ovpn  &
    # wait until tun0 becomes ready
    while [ ! -f /sys/class/net/tun0/operstate ]
    do
      sleep 1
    done
    IPADDR=$( ip a s tun0 | awk '/inet.*brd/ {print $2}' )
    # get route network we need to add to local route table
    IFS=. read -r i1 i2 i3 i4 <<< $IPADDR
    IFS=. read -r m1 m2 m3 m4 <<< "255.255.0.0"
    ROUTE=$(printf "%d.%d.%d.%d/16\n" "$((i1 & m1))" "$((i2 & m2))" "$((i3 & m3))" "$((i4 & m4))")
    echo "Adding route $ROUTE to tun0"
    sudo ip route add $ROUTE dev tun0
    ;;

status)
    if [[ "$#" -ne 2 ]]; then
        usage
    fi
    ps aux | grep -v grep | grep "openvpn"
    ;;
stop)
    if [[ "$#" -ne 2 ]]; then
        usage
    fi
    ps aux | grep -v grep | grep "openvpn" | awk {'print $2'} | xargs sudo kill
    ;;
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
