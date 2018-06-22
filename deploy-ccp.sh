#!/bin/bash

set -eu

# run helm, need a pre-existing cluster
# src=$(mktemp -d)
# cd $src
# export GOPATH=`pwd`
# go get github.com/jim-minter/azure-helm
# cd azure-helm
# make push

# source azure credentials
source ~/.azure/credentials

# pretty sure there is a trick with sed to do the same thing
cat >./newline.go <<EOF
package main

import (
  "bufio"
  "fmt"
  "os"
  "strings"
)

func main() {
  var out string
  scanner := bufio.NewScanner(os.Stdin)
  for scanner.Scan() {
    out += scanner.Text() + "\\\n"
  }
  if err := scanner.Err(); err != nil {
    fmt.Fprintln(os.Stderr, "reading standard input:", err)
  } else {
    fmt.Fprintln(os.Stdout, strings.TrimSuffix(out, "\\\n"))
  }
}
EOF

# acs-engine inputs
CRT=$(oc get secret etc-origin-master -o jsonpath='{.data.ca\.crt}' | base64 -d | go run newline.go)
KEY=$(oc get secret etc-origin-master -o jsonpath='{.data.ca\.key}' | base64 -d | go run newline.go)
HOSTNAME=$(oc get secret etc-origin-master -o jsonpath='{.data.master-config\.yaml}' | base64 -d | grep masterURL | awk '{print $2}')
RESOURCE_GROUP=$(oc get secret etc-origin-cloudprovider -o jsonpath='{.data.azure\.conf}' | base64 -d | grep resourceGroup | awk '{print $2}')
LOCATION=$(oc get secret etc-origin-cloudprovider -o jsonpath='{.data.azure\.conf}' | base64 -d | grep location | awk '{print $2}')

# We shouldn't need an SSH key: https://github.com/Azure/acs-engine/issues/3339
ssh-keygen -t rsa -f dummy -b 4096 -N dummy

# TODO: dnsPrefix is redundant and can go away: https://github.com/Azure/acs-engine/issues/3338
cat >./openshift-model.json <<EOF
{
  "apiVersion": "vlabs",
  "name": "",
  "properties": {
    "dnsPrefix": "agentsonlyapi",
    "fqdn": "${HOSTNAME#https://}",
    "orchestratorProfile": {
      "orchestratorType": "OpenShift",
      "orchestratorVersion": "unstable",
      "openshiftConfig": {}
    },
    "azProfile": {
      "tenantId": "$AZURE_TENANT_ID",
      "subscriptionId": "$AZURE_SUBSCRIPTION_ID",
      "resourceGroup": "$RESOURCE_GROUP",
      "location": "$LOCATION"
    },
    "agentPoolProfiles": [
      {
        "availabilityProfile": "AvailabilitySet",
        "count": 1,
        "imageReference": {
          "name": "centos7-3.10-201806051434",
          "resourceGroup": "images"
        },
        "name": "compute",
        "storageProfile": "ManagedDisks",
        "vmSize": "Standard_D4s_v3"
      },
      {
        "availabilityProfile": "AvailabilitySet",
        "count": 1,
        "imageReference": {
          "name": "centos7-3.10-201806051434",
          "resourceGroup": "images"
        },
        "name": "infra",
        "role": "infra",
        "storageProfile": "ManagedDisks",
        "vmSize": "Standard_D4s_v3"
      }
    ],
    "linuxProfile": {
      "adminUsername": "cloud-user",
      "ssh": {
        "publicKeys": [
          {
            "keyData": "$(cat dummy.pub)"
          }
        ]
      }
    },
    "servicePrincipalProfile": {
      "clientId": "$AZURE_CLIENT_ID",
      "secret": "$AZURE_CLIENT_SECRET"
    },
    "certificateProfile": {
      "caCertificate": "$(echo $CRT)",
      "caPrivateKey": "$(echo $KEY)"
    }
  }
}
EOF

# cleanup
rm dummy dummy.pub

# run acs-engine with https://github.com/Azure/acs-engine/pull/3126
acs-engine deploy -f -g $RESOURCE_GROUP -l $LOCATION --subscription-id $AZURE_SUBSCRIPTION_ID openshift-model.json
