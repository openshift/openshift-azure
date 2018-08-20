#!/bin/bash -e

usage() {
    cat <<EOF >&2
usage:

$0 app-create hostname resourcegroup
$0 app-delete hostname

EOF
    exit 1
}

case "$1" in
app-create)
    if [[ "$#" -ne 3 ]]; then usage; fi

    AZURE_AAD_CLIENT_SECRET=$(uuidgen)
    AZURE_AAD_CLIENT_ID=$(az ad app create \
        --display-name "$2" \
        --homepage "https://$2/" \
        --identifier-uris "https://$2/" \
        --key-type password \
        --password "$AZURE_AAD_CLIENT_SECRET" \
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
)

    az ad sp create --id $AZURE_AAD_CLIENT_ID >/dev/null

    success=
    for ((i=0; i<12; i++)); do
        if az role assignment create -g $3 --assignee $AZURE_AAD_CLIENT_ID --role contributor &>/dev/null; then
            success=true
            break
        fi
        sleep 5
    done
    if [[ -z "$success" ]]; then
        echo 'error: failed to assign contributor role to SP' >&2
        exit 1
    fi

    cat <<EOF
AZURE_AAD_CLIENT_ID=$AZURE_AAD_CLIENT_ID
AZURE_AAD_CLIENT_SECRET=$AZURE_AAD_CLIENT_SECRET
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
