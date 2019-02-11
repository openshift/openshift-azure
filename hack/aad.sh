#!/bin/bash -e

usage() {
    cat <<EOF >&2
usage:

$0 app-create name callbackurl
$0 app-delete appId
$0 app-update appId callbackurl

Examples:
aad.sh app-create test-app https://openshift.test.osadev.cloud/oauth2callback/Azure%20AD
aad.sh app-delete 76a604b8-0896-4ab7-9ef4-xxxxxxxxxx
aad.sh app-update 76a604b8-0896-4ab7-9ef4-xxxxxxxxxx https://openshift.newtest.osadev.cloud/oauth2callback/Azure%20AD


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
    if [[ "$#" -ne 3 ]]; then usage; fi

    AZURE_AAD_CLIENT_SECRET=$(uuidgen)
    AZURE_AAD_CLIENT_ID=$(az ad app create \
        --display-name "$2" \
        --homepage "$3" \
        --identifier-uris "$3" \
        --key-type password \
        --password "$AZURE_AAD_CLIENT_SECRET" \
        --query appId \
        --reply-urls "$3" \
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

    cat <<EOF
AZURE_AAD_CLIENT_ID=$AZURE_AAD_CLIENT_ID

EOF

cat >&2 <<EOF
Note: For the application to work, an Organization Administrator needs to grant permissions first.
      Once it is approved, it can be reused for other clusters using app-update functionality

      To use this AAD application with OpenShift cluster value below must be present in your env before creating the cluster
      export AZURE_AAD_CLIENT_ID=$AZURE_AAD_CLIENT_ID
EOF
    ;;

app-update)
    if [[ "$#" -ne 3 ]]; then usage; fi
    AZURE_AAD_CLIENT_SECRET=$(uuidgen)
    az ad app update --id $2 --reply-urls "$3" --key-type password --password $AZURE_AAD_CLIENT_SECRET

cat <<EOF
AZURE_AAD_CLIENT_ID=$2
AZURE_AAD_CLIENT_SECRET=$AZURE_AAD_CLIENT_SECRET
EOF
    ;;

app-delete)
    if [[ "$#" -ne 2 ]]; then usage; fi
    az ad app delete --id $2
    ;;

*)
    usage
    ;;

esac
