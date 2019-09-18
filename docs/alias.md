# ARO usefull alias


## Upload, download private secrets
```
alias azure-downloads-private-secrets="oc extract secret/cluster-secrets-azure --to=./secrets --confirm -n azure-private"
alias azure-upload-private-secrets="oc create secret generic cluster-secrets-azure \
        --from-file=./private-secrets/vpn-caCert.pem \
        --from-file=./private-secrets/vpn-caKey.pem \
        --from-file=./private-secrets/vpn-clientCert.pem \
        --from-file=./private-secrets/vpn-clientKey.pem \
        --from-file=./private-secrets/vpn-client.p12 \
        --dry-run -o yaml | oc apply -n azure-private -f - "
```

## Upload, download team secrets
```
alias azure-downloads-secrets="oc extract secret/cluster-secrets-azure --to=./secrets --confirm -n azure"
alias azure-upload-private-secrets="oc create secret generic cluster-secrets-azure \
        --from-file=./secrets/certs.yaml \
        --from-file=./secrets/logging-int.cert \
        --from-file=./secrets/metrics-int.cert \
        --from-file=./secrets/secret \
        --from-file=./secrets/rh-docker-pull-secret \
        --from-file=./secrets/acr-docker-pull-secert \
        --from-file=./secrets/logging-int.key \
        --from-file=./secrets/metrics-int.key \
        --from-file=./secrets/ssh-privatekey \
        --from-file=./secrets/pull-secret.txt \
        --from-file=./secrets/client-key.pem \
        --from-file=./secrets/client-cert.pem \
        --from-file=./secrets/vpn-rootCA.der \
        --from-file=./secrets/vpn-clientKey.pem \
        --from-file=./secrets/vpn-clientCert.pem \
        --from-file=./secrets/vpn-westeurope.ovpn \
        --from-file=./secrets/vpn-eastus.ovpn \
        --from-file=./secrets/vpn-australiasoutheast.ovpn \
        --dry-run -o yaml | oc apply -n azure -f - "
```

## Acitivity logs for the cluster
```
alias cluster-logs="az monitor activity-log list -g $RESOURCEGROUP --offset 7d --query "[].[eventTimestamp,submissionTimestamp,level,resourceId,eventName.value,operationName.value,status.value]" --max-events 400 -o tsv | column -t
```