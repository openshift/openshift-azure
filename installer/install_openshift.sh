#! /bin/bash -e 

echo "Welcome to the Azure Red Hat OpenShift installation script. This script will require some variables"  
read -p "What will be the ResourceGroup be called? " RESGR_NAME
read -p "What will be the OpenShift cluster's name? " CLUSTER_NAME
read -p "Where will the cluster be located? " LOCATION
read -p "What is the name of the Active Directory admin group? " ADMINGROUP
read -p "What is the Azure AD Application (Client ID)? " APPID
read -p "What is the Azure AD Application secret? " SECRET
echo "That is all the information needed. From here, installation will automatically progress. Thank you for using Azure Red Hat OpenShift."

export TENANT=$(az account show --query name -o tsv)
export TOKEN=$(az account get-access-token --query 'accessToken' -o tsv)
export SUBID=$(az account list --all --query "[?name=='$TENANT'].id" -o tsv)
export TENANT_ID=$(az account list --all --query "[?name=='$TENANT'].tenantId" -o tsv)
export ADMIN_GROUP=$(az ad group list --query "[?displayName=='$ADMINGROUP'].objectId" -o tsv)

echo "Creating Resource Group"
az group create --name $RESGR_NAME --location $LOCATION
az group wait --name $RESGR_NAME --subscription $SUBID --created

curl -v -X PUT -H 'Content-Type: application/json; charset=utf-8' \
-H 'Authorization: Bearer '$TOKEN'' \
-d '{ "location": "'$LOCATION'", "tags": { "tier": "production", "archv2": "" },  "properties": { "openShiftVersion": "v3.11", "networkProfile": { "vnetCidr": "10.0.0.0/8" }, "masterPoolProfile": { "name": "master", "count": 3, "vmSize": "Standard_D4s_v3", "osType": "Linux", "subnetCidr": "10.0.0.0/24" }, "agentPoolProfiles": [ { "name": "infra", "role": "infra", "count": 3, "vmSize": "Standard_D4s_v3", "osType": "Linux", "subnetCidr": "10.0.0.0/24" }, { "name": "compute", "role": "compute", "count": 1, "vmSize": "Standard_D4s_v3", "osType": "Linux", "subnetCidr": "10.0.0.0/24" } ], "routerProfiles": [ { "name": "default" } ], "authProfile": { "identityProviders": [ { "name": "Azure AD", "provider": { "kind": "AADIdentityProvider", "clientId": "'$APPID'", "secret": "'$SECRET'", "tenantId": "'$TENANT_ID'", "customerAdminGroupId": "'$ADMIN_GROUP'" }}]}}}' \
https://management.azure.com/subscriptions/$SUBID/resourceGroups/$RESGR_NAME/providers/Microsoft.ContainerService/openShiftManagedClusters/$CLUSTER_NAME?api-version=2019-04-30

echo "Creating Cluster"
az openshift create --resource-group $RESGR_NAME --name $CLUSTER_NAME -l $LOCATION --aad-client-app-id $APPID --aad-client-app-secret $SECRET --aad-tenant-id $TENANT_ID --customer-admin-group-id $ADMIN_GROUP

URI=$(az openshift show -n $RESGR_NAME -g $CLUSTER_NAME -o yaml --query publicHostname -o tsv)

az ad app update --id $APPID --reply-urls https://$URI/oauth2callback/Azure%20AD

echo "Updating Azure AD app with callback URL" 

az openshift list -g $RESGR_NAME | grep provisioningState

az openshift wait -n $CLUSTER_NAME --created

echo "Azure Red Hat OpenShift cluster successfully created"