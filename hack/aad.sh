#!/bin/bash -e

usage() {
    cat <<EOF >&2
usage:

$0 app-create name owner

Examples:
$0 app-create user-$USER-aad jminter-team-shared

EOF
    exit 1
}

# Meaning of the below magic numbers
# "00000003-0000-0000-c000-000000000000" -> microsoft graph
# "5b567255-7703-4780-807c-7be8301ae99b" -> read all groups
# "37f7f235-527c-4136-accd-4a02d197296e" -> sign users in
# "type": "Role" -> application permission
# "type": "Scope" -> delegated permission

case "$1" in
app-create)
    if [[ "$#" -ne 3 ]]; then
        usage
    fi

    OWNER_ID=$(az ad sp list --display-name "$3" --query [0].objectId --output tsv)
    if [[ "$OWNER_ID" == "" ]]; then
        echo "owner $3 not found" >&2
        exit 1
    fi

    AZURE_AAD_CLIENT_SECRET=$(uuidgen)
    AZURE_AAD_CLIENT_ID=$(az ad app create \
        --display-name "$2" \
        --homepage http://localhost/ \
        --identifier-uris http://localhost/ \
        --key-type password \
        --password "$AZURE_AAD_CLIENT_SECRET" \
        --query appId \
        --reply-urls http://localhost/ \
        --required-resource-accesses @- <<'EOF' | tr -d '"'
[
    {
      "resourceAppId": "00000003-0000-0000-c000-000000000000",
      "resourceAccess": [
        {
          "id": "5b567255-7703-4780-807c-7be8301ae99b",
          "type": "Role"
        },
        {
          "id": "37f7f235-527c-4136-accd-4a02d197296e",
          "type": "Scope"
        }
      ]
    }
]
EOF
)

    az ad app owner add --id $AZURE_AAD_CLIENT_ID --owner-object-id $OWNER_ID

    cat >&2 <<EOF
Note: ask an administrator to grant your application's permissions.  Until this
      is done the application will not work.

To use this application with an OpenShift cluster, add the following line to
your env file and source it:

export AZURE_AAD_CLIENT_ID=$AZURE_AAD_CLIENT_ID
EOF
    ;;

*)
    usage
    ;;

esac
