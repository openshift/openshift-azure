#!/bin/bash -e

usage() {
    cat <<EOF >&2
usage:

$0 app-create hostname password
$0 app-delete hostname

EOF
    exit 1
}

case "$1" in
app-create)
    if [[ "$#" -ne 3 ]]; then usage; fi

    az ad app create \
        --display-name "$2" \
        --homepage "https://$2/" \
        --identifier-uris "https://$2/" \
        --key-type password \
        --password "$3" \
        --query appId \
        --reply-urls "https://$2/oauth2callback/Azure%20AD" \
        --required-resource-accesses @- <<'EOF' | tr -d '"'
[
    {
        "resourceAppId": "00000002-0000-0000-c000-000000000000",
        "resourceAccess": [
            {
                "id": "311a71cc-e848-46a1-bdf8-97ff7156d8e6",
                "type": "Scope"
            }
        ]
    }
]
EOF
    ;;

app-delete)
    if [[ "$#" -ne 2 ]]; then usage; fi

    ID=$(az ad app list --query "[?homepage=='https://$2/'].appId | [0]" | tr -d '"')
    if [[ -n "$ID" ]]; then
        az ad app delete --id "$ID"
    fi
    ;;

*)
    usage
    ;;

esac
