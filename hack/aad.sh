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
# "00000002-0000-0000-c000-000000000000" -> AAD Graph API
# "5778995a-e1bf-45b8-affa-663a9f3f4d04" -> Read directory data
# "311a71cc-e848-46a1-bdf8-97ff7156d8e6" -> Sign in and read user profile
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
      "resourceAppId": "00000002-0000-0000-c000-000000000000",
      "resourceAccess": [
        {
          "id": "5778995a-e1bf-45b8-affa-663a9f3f4d04",
          "type": "Role"
        },
        {
          "id": "311a71cc-e848-46a1-bdf8-97ff7156d8e6",
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
export AZURE_AAD_CLIENT_SECRET=$AZURE_AAD_CLIENT_SECRET
EOF
    ;;

*)
    usage
    ;;

esac
