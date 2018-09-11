#!/bin/bash -ex

# TODO: /etc/dnsmasq.d/origin-upstream-dns.conf is currently hardcoded; it
# probably shouldn't be

SERVICE_TYPE=origin
if [ -f "/etc/sysconfig/atomic-openshift-node" ]; then
    SERVICE_TYPE=atomic-openshift
fi

if ! grep /var/lib/docker /etc/fstab; then
  mkfs.xfs -f /dev/disk/azure/resource-part1
  echo '/dev/disk/azure/resource-part1  /var/lib/docker  xfs  grpquota  0 0' >>/etc/fstab
  systemctl stop docker.service
  mount /var/lib/docker
  restorecon -R /var/lib/docker
  cat >/etc/docker/daemon.json <<'EOF'
{
  "log-driver": "journald"
}
EOF
  systemctl start docker.service
fi

# TODO: consider fact that /dev/disk/azure/scsi1/lun0 is currently hardcoded;
# partition /dev/disk/azure/scsi1/lun0; consider future strategy for resizes if
# needed
if ! grep /var/lib/etcd /etc/fstab; then
  mkfs.xfs /dev/disk/azure/scsi1/lun0 || true
  echo '/dev/disk/azure/scsi1/lun0  /var/lib/etcd  xfs  defaults  0 0' >>/etc/fstab
  mount /var/lib/etcd
  restorecon -R /var/lib/etcd
fi

echo "BOOTSTRAP_CONFIG_NAME=node-config-master" >>/etc/sysconfig/${SERVICE_TYPE}-node

sed -i -e "s#DEBUG_LOGLEVEL=2#DEBUG_LOGLEVEL=4#" /etc/sysconfig/${SERVICE_TYPE}-node

for dst in tcp,2380; do
#for dst in tcp,2379 tcp,2380 tcp,8443 tcp,8444 tcp,8053 udp,8053 tcp,9090; do
	proto=${dst%%,*}
	port=${dst##*,}
	iptables -A OS_FIREWALL_ALLOW -p $proto -m state --state NEW -m $proto --dport $port -j ACCEPT
done

iptables-save >/etc/sysconfig/iptables

rm -rf /etc/etcd/* /etc/origin/master/*

base64 -d <<< {{ CertAsBytes .Config.Certificates.EtcdCa.Cert | Base64Encode }} >/etc/etcd/ca.crt
base64 -d <<< {{ CertAsBytes .Config.Certificates.EtcdServer.Cert | Base64Encode }} >/etc/etcd/server.crt
base64 -d <<< {{ PrivateKeyAsBytes .Config.Certificates.EtcdServer.Key | Base64Encode }} >/etc/etcd/server.key
base64 -d <<< {{ CertAsBytes .Config.Certificates.EtcdPeer.Cert | Base64Encode }} >/etc/etcd/peer.crt
base64 -d <<< {{ PrivateKeyAsBytes .Config.Certificates.EtcdPeer.Key | Base64Encode }} >/etc/etcd/peer.key

base64 -d <<< {{ YamlMarshal .Config.AdminKubeconfig | Base64Encode }} >/etc/origin/node/node.kubeconfig
base64 -d <<< {{ CertAsBytes .Config.Certificates.Ca.Cert | Base64Encode }} >/etc/origin/node/ca.crt

mkdir -p /etc/origin/master/named
base64 -d <<< {{ CertAsBytes .Config.Certificates.EtcdCa.Cert | Base64Encode }} >/etc/origin/master/master.etcd-ca.crt
base64 -d <<< {{ CertAsBytes .Config.Certificates.Ca.Cert | Base64Encode }} >/etc/origin/master/ca.crt
base64 -d <<< {{ PrivateKeyAsBytes .Config.Certificates.Ca.Key | Base64Encode }} >/etc/origin/master/ca.key
base64 -d <<< {{ CertAsBytes .Config.Certificates.OpenshiftConsole.Cert | Base64Encode }} >/etc/origin/master/named/console.crt
base64 -d <<< {{ PrivateKeyAsBytes .Config.Certificates.OpenshiftConsole.Key | Base64Encode }} >/etc/origin/master/named/console.key
base64 -d <<< {{ CertAsBytes .Config.Certificates.FrontProxyCa.Cert | Base64Encode }} >/etc/origin/master/front-proxy-ca.crt
base64 -d <<< {{ CertAsBytes .Config.Certificates.ServiceSigningCa.Cert | Base64Encode }} >/etc/origin/master/service-signer.crt
base64 -d <<< {{ PrivateKeyAsBytes .Config.Certificates.ServiceSigningCa.Key | Base64Encode }} >/etc/origin/master/service-signer.key
base64 -d <<< {{ CertAsBytes .Config.Certificates.EtcdClient.Cert | Base64Encode }} >/etc/origin/master/master.etcd-client.crt
base64 -d <<< {{ PrivateKeyAsBytes .Config.Certificates.EtcdClient.Key | Base64Encode }} >/etc/origin/master/master.etcd-client.key
base64 -d <<< {{ CertAsBytes .Config.Certificates.AggregatorFrontProxy.Cert | Base64Encode }} >/etc/origin/master/aggregator-front-proxy.crt
base64 -d <<< {{ PrivateKeyAsBytes .Config.Certificates.AggregatorFrontProxy.Key | Base64Encode }} >/etc/origin/master/aggregator-front-proxy.key
base64 -d <<< {{ CertAsBytes .Config.Certificates.MasterKubeletClient.Cert | Base64Encode }} >/etc/origin/master/master.kubelet-client.crt
base64 -d <<< {{ PrivateKeyAsBytes .Config.Certificates.MasterKubeletClient.Key | Base64Encode }} >/etc/origin/master/master.kubelet-client.key
base64 -d <<< {{ CertAsBytes .Config.Certificates.MasterProxyClient.Cert | Base64Encode }} >/etc/origin/master/master.proxy-client.crt
base64 -d <<< {{ PrivateKeyAsBytes .Config.Certificates.MasterProxyClient.Key | Base64Encode }} >/etc/origin/master/master.proxy-client.key
base64 -d <<< {{ CertAsBytes .Config.Certificates.MasterServer.Cert | Base64Encode }} >/etc/origin/master/master.server.crt
base64 -d <<< {{ PrivateKeyAsBytes .Config.Certificates.MasterServer.Key | Base64Encode }} >/etc/origin/master/master.server.key
base64 -d <<< {{ PublicKeyAsBytes .Config.ServiceAccountKey.PublicKey | Base64Encode }} >/etc/origin/master/serviceaccounts.public.key
base64 -d <<< {{ PrivateKeyAsBytes .Config.ServiceAccountKey | Base64Encode }} >/etc/origin/master/serviceaccounts.private.key
base64 -d <<< {{ .Config.HtPasswd | Base64Encode }} >/etc/origin/master/htpasswd
base64 -d <<< {{ YamlMarshal .Config.AdminKubeconfig | Base64Encode }} >/etc/origin/master/admin.kubeconfig
base64 -d <<< {{ YamlMarshal .Config.MasterKubeconfig | Base64Encode }} >/etc/origin/master/openshift-master.kubeconfig

cat >/etc/etcd/etcd.conf <<EOF
ETCD_ADVERTISE_CLIENT_URLS=https://$(hostname):2379
ETCD_CERT_FILE=/etc/etcd/server.crt
ETCD_CLIENT_CERT_AUTH=true
ETCD_DATA_DIR=/var/lib/etcd
ETCD_ELECTION_TIMEOUT=2500
ETCD_HEARTBEAT_INTERVAL=500
ETCD_INITIAL_ADVERTISE_PEER_URLS=https://$(hostname):2380
ETCD_INITIAL_CLUSTER=master-000000=https://master-000000:2380,master-000001=https://master-000001:2380,master-000002=https://master-000002:2380
ETCD_KEY_FILE=/etc/etcd/server.key
ETCD_LISTEN_CLIENT_URLS=https://0.0.0.0:2379
ETCD_LISTEN_PEER_URLS=https://0.0.0.0:2380
ETCD_NAME=$(hostname)
ETCD_PEER_CERT_FILE=/etc/etcd/peer.crt
ETCD_PEER_CLIENT_CERT_AUTH=true
ETCD_PEER_KEY_FILE=/etc/etcd/peer.key
ETCD_PEER_TRUSTED_CA_FILE=/etc/etcd/ca.crt
ETCD_QUOTA_BACKEND_BYTES=4294967296
ETCD_TRUSTED_CA_FILE=/etc/etcd/ca.crt
EOF

cp /etc/origin/node/ca.crt /etc/origin/node/client-ca.crt
cp /etc/origin/node/ca.crt /etc/pki/ca-trust/source/anchors/openshift-ca.crt
update-ca-trust

cat >/etc/origin/master/master-config.yaml <<EOF
admissionConfig:
  pluginConfig:
    AlwaysPullImages:
      configuration:
        kind: DefaultAdmissionConfig
        apiVersion: v1
        disable: false
    BuildDefaults:
      configuration:
        apiVersion: v1
        kind: BuildDefaultsConfig
    BuildOverrides:
      configuration:
        apiVersion: v1
        kind: BuildOverridesConfig
    PodPreset:
      configuration:
        apiVersion: v1
        kind: DefaultAdmissionConfig
    openshift.io/ImagePolicy:
      configuration:
        apiVersion: v1
        executionRules:
        - matchImageAnnotations:
          - key: images.openshift.io/deny-execution
            value: "true"
          name: execution-denied
          onResources:
          - resource: pods
          - resource: builds
          reject: true
          skipOnResolutionFailure: true
        kind: ImagePolicyConfig
aggregatorConfig:
  proxyClientInfo:
    certFile: aggregator-front-proxy.crt
    keyFile: aggregator-front-proxy.key
apiLevels:
- v1
apiVersion: v1
authConfig:
  requestHeader:
    clientCA: front-proxy-ca.crt
    clientCommonNames:
    - aggregator-front-proxy
    extraHeaderPrefixes:
    - X-Remote-Extra-
    groupHeaders:
    - X-Remote-Group
    usernameHeaders:
    - X-Remote-User
controllerConfig:
  election:
    lockName: openshift-master-controllers
  serviceServingCert:
    signer:
      certFile: service-signer.crt
      keyFile: service-signer.key
controllers: "*"
corsAllowedOrigins:
dnsConfig:
  bindAddress: 0.0.0.0:8053
  bindNetwork: tcp4
etcdClientInfo:
  ca: master.etcd-ca.crt
  certFile: master.etcd-client.crt
  keyFile: master.etcd-client.key
  urls:
  - https://$(hostname):2379
etcdStorageConfig:
  kubernetesStoragePrefix: kubernetes.io
  kubernetesStorageVersion: v1
  openShiftStoragePrefix: openshift.io
  openShiftStorageVersion: v1
imageConfig:
  format: {{ .Config.ImageConfigFormat | escape }}
imagePolicyConfig:
  internalRegistryHostname: docker-registry.default.svc:5000
kind: MasterConfig
kubeletClientInfo:
  ca: ca.crt
  certFile: master.kubelet-client.crt
  keyFile: master.kubelet-client.key
  port: 10250
kubernetesMasterConfig:
  apiServerArguments:
    cloud-config:
    - /etc/origin/cloudprovider/azure.conf
    cloud-provider:
    - azure
    runtime-config:
    - settings.k8s.io/v1alpha1=true
    storage-backend:
    - etcd3
    storage-media-type:
    - application/vnd.kubernetes.protobuf
  controllerArguments:
    cloud-config:
    - /etc/origin/cloudprovider/azure.conf
    cloud-provider:
    - azure
    cluster-signing-cert-file:
    - /etc/origin/master/ca.crt
    cluster-signing-key-file:
    - /etc/origin/master/ca.key
  masterIP: 127.0.0.1
  proxyClientInfo:
    certFile: master.proxy-client.crt
    keyFile: master.proxy-client.key
  schedulerConfigFile: /etc/origin/master/scheduler.json
  servicesSubnet: 172.30.0.0/16
masterClients:
  openshiftLoopbackClientConnectionOverrides:
    acceptContentTypes: application/vnd.kubernetes.protobuf,application/json
    burst: 600
    contentType: application/vnd.kubernetes.protobuf
    qps: 300
  openshiftLoopbackKubeConfig: openshift-master.kubeconfig
masterPublicURL: {{ print "https://" .ContainerService.Properties.PublicHostname | quote }}
networkConfig:
  clusterNetworks:
  - cidr: 10.128.0.0/14
    hostSubnetLength: 9
  externalIPNetworkCIDRs:
  - 0.0.0.0/0
  networkPluginName: redhat/openshift-ovs-subnet
  serviceNetworkCIDR: 172.30.0.0/16
oauthConfig:
  assetPublicURL: {{ print "https://" .ContainerService.Properties.PublicHostname "/console/" | quote }}
  grantConfig:
    method: auto
  identityProviders:
  - login: true
    mappingMethod: claim
    name: {{ (index .ContainerService.Properties.AuthProfile.IdentityProviders 0).Name | quote }}
    provider:
      apiVersion: v1
      claims:
        email:
        - email
        id:
        - sub
        name:
        - name
        preferredUsername:
        - unique_name
      clientID: {{ (index .ContainerService.Properties.AuthProfile.IdentityProviders 0).Provider.ClientID | quote }}
      clientSecret: {{ (index .ContainerService.Properties.AuthProfile.IdentityProviders 0).Provider.Secret | quote }}
      kind: OpenIDIdentityProvider
      urls:
        authorize: {{ print "https://login.microsoftonline.com/" (index .ContainerService.Properties.AuthProfile.IdentityProviders 0).Provider.TenantID "/oauth2/authorize" | quote }}
        token: {{ print "https://login.microsoftonline.com/" (index .ContainerService.Properties.AuthProfile.IdentityProviders 0).Provider.TenantID "/oauth2/token" | quote }}
  - challenge: true
    login: true
    mappingMethod: claim
    name: Local password
    provider:
      apiVersion: v1
      file: /etc/origin/master/htpasswd
      kind: HTPasswdPasswordIdentityProvider
  masterCA: ca.crt
  masterPublicURL: {{ print "https://" .ContainerService.Properties.PublicHostname | quote }}
  masterURL: {{ print "https://" .ContainerService.Properties.FQDN | quote }}
  sessionConfig:
    sessionMaxAgeSeconds: 3600
    sessionName: ssn
    sessionSecretsFile: /etc/origin/master/session-secrets.yaml
  tokenConfig:
    accessTokenMaxAgeSeconds: 86400
    authorizeTokenMaxAgeSeconds: 500
projectConfig:
  defaultNodeSelector: node-role.kubernetes.io/compute=true
  securityAllocator:
    mcsAllocatorRange: s0:/2
    mcsLabelsPerProject: 5
    uidAllocatorRange: 1000000000-1999999999/10000
routingConfig:
  subdomain: {{ (index .ContainerService.Properties.RouterProfiles 0).PublicSubdomain | quote }}
serviceAccountConfig:
  managedNames:
  - default
  - builder
  - deployer
  masterCA: ca.crt
  privateKeyFile: serviceaccounts.private.key
  publicKeyFiles:
  - serviceaccounts.public.key
servingInfo:
  bindAddress: 0.0.0.0:443
  bindNetwork: tcp4
  certFile: master.server.crt
  clientCA: ca.crt
  keyFile: master.server.key
  maxRequestsInFlight: 500
  requestTimeoutSeconds: 3600
  namedCertificates:
   - certFile: /etc/origin/master/named/console.crt
     keyFile: /etc/origin/master/named/console.key
     names:
      - {{ .ContainerService.Properties.PublicHostname | quote }}
volumeConfig:
  dynamicProvisioningEnabled: true
EOF

cat >/etc/origin/master/session-secrets.yaml <<'EOF'
apiVersion: v1
kind: SessionSecrets
secrets:
- authentication: {{ .Config.SessionSecretAuth | Base64Encode | quote }}
  encryption: {{ .Config.SessionSecretEnc | Base64Encode | quote }}
EOF

cat >/etc/origin/master/scheduler.json <<'EOF'
{
  "apiVersion": "v1",
  "kind": "Policy",
  "predicates": [
    {
      "name": "NoVolumeZoneConflict"
    },
    {
      "name": "MaxEBSVolumeCount"
    },
    {
      "name": "MaxGCEPDVolumeCount"
    },
    {
      "name": "MaxAzureDiskVolumeCount"
    },
    {
      "name": "MatchInterPodAffinity"
    },
    {
      "name": "NoDiskConflict"
    },
    {
      "name": "GeneralPredicates"
    },
    {
      "name": "PodToleratesNodeTaints"
    },
    {
      "name": "CheckNodeMemoryPressure"
    },
    {
      "name": "CheckNodeDiskPressure"
    },
    {
      "name": "CheckVolumeBinding"
    },
    {
      "argument": {
        "serviceAffinity": {
          "labels": [
            "region"
          ]
        }
      },
      "name": "Region"
    }
  ],
  "priorities": [
    {
      "name": "SelectorSpreadPriority",
      "weight": 1
    },
    {
      "name": "InterPodAffinityPriority",
      "weight": 1
    },
    {
      "name": "LeastRequestedPriority",
      "weight": 1
    },
    {
      "name": "BalancedResourceAllocation",
      "weight": 1
    },
    {
      "name": "NodePreferAvoidPodsPriority",
      "weight": 10000
    },
    {
      "name": "NodeAffinityPriority",
      "weight": 1
    },
    {
      "name": "TaintTolerationPriority",
      "weight": 1
    },
    {
      "argument": {
        "serviceAntiAffinity": {
          "label": "zone"
        }
      },
      "name": "Zone",
      "weight": 2
    }
  ]
}
EOF

echo 'nameserver 168.63.129.16' >/etc/origin/node/resolv.conf
mkdir -p /etc/origin/cloudprovider

cat >/etc/origin/cloudprovider/azure.conf <<'EOF'
{{ .Config.CloudProviderConf | String }}
EOF

# TODO: investigate the --manifest-url Kubelet parameter and see if it might
# help us at all
cat >/etc/origin/node/pods/etcd.yaml <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: etcd
  namespace: kube-system
spec:
  containers:
  - args:
    - |
      set -a
      . /etc/etcd/etcd.conf
      exec etcd
    command:
    - /bin/sh
    - -c
    image: {{ .Config.MasterEtcdImage | quote }}
    imagePullPolicy: Always
    livenessProbe:
      exec:
        command:
        - etcdctl
        - --ca-file
        - /etc/etcd/ca.crt
        - --cert-file
        - /etc/etcd/peer.crt
        - --key-file
        - /etc/etcd/peer.key
        - --endpoints
        - https://$(hostname):2379
        - cluster-health
      initialDelaySeconds: 45
    name: etcd
    securityContext:
      privileged: true
    volumeMounts:
    - mountPath: /etc/etcd
      name: etcd-config
      readOnly: true
    - mountPath: /var/lib/etcd
      name: etcd-data
    workingDir: /var/lib/etcd
  hostNetwork: true
  volumes:
  - hostPath:
      path: /etc/etcd
    name: etcd-config
  - hostPath:
      path: /var/lib/etcd
    name: etcd-data
EOF

cat >/etc/origin/node/pods/api.yaml <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  labels:
    openshift.io/component: api
  name: api
  namespace: kube-system
spec:
  containers:
  - args:
    - start
    - master
    - api
    - --config=/etc/origin/master/master-config.yaml
    - --loglevel=4
    command:
    - openshift
    image: {{ .Config.ControlPlaneImage | quote }}
    imagePullPolicy: Always
    livenessProbe:
      httpGet:
        path: healthz
        port: 443
        scheme: HTTPS
      initialDelaySeconds: 45
      timeoutSeconds: 10
    name: api
    readinessProbe:
      httpGet:
        path: healthz/ready
        port: 443
        scheme: HTTPS
      initialDelaySeconds: 10
      timeoutSeconds: 10
    securityContext:
      privileged: true
    volumeMounts:
    - mountPath: /etc/origin/master
      name: master-config
      readOnly: true
    - mountPath: /etc/origin/cloudprovider
      name: master-cloud-provider
      readOnly: true
  hostNetwork: true
  volumes:
  - hostPath:
      path: /etc/origin/master
    name: master-config
  - hostPath:
      path: /etc/origin/cloudprovider
    name: master-cloud-provider
EOF

cat >/etc/origin/node/pods/controllers.yaml <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: controllers
  namespace: kube-system
spec:
  containers:
  - args:
    - start
    - master
    - controllers
    - --config=/etc/origin/master/master-config.yaml
    - --listen=https://0.0.0.0:444
    - --loglevel=4
    command:
    - openshift
    image: {{ .Config.ControlPlaneImage | quote }}
    imagePullPolicy: Always
    livenessProbe:
      httpGet:
        path: healthz
        port: 444
        scheme: HTTPS
    name: controllers
    securityContext:
      privileged: true
    volumeMounts:
    - mountPath: /etc/origin/master
      name: master-config
      readOnly: true
    - mountPath: /etc/origin/cloudprovider
      name: master-cloud-provider
      readOnly: true
  hostNetwork: true
  volumes:
  - hostPath:
      path: /etc/origin/master
    name: master-config
  - hostPath:
      path: /etc/origin/cloudprovider
    name: master-cloud-provider
EOF

if [[ "$(hostname)" == "master-000000" ]]; then
  cat >/etc/origin/node/pods/sync.yaml <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: sync
  namespace: kube-system
spec:
  containers:
  - image: {{ .Config.SyncImage | quote }}
    imagePullPolicy: Always
    name: sync
    securityContext:
      privileged: true
    volumeMounts:
    - mountPath: /_data/_out
      name: master-cloud-provider
      readOnly: true
  hostNetwork: true
  volumes:
  - hostPath:
      path: /etc/origin/cloudprovider
    name: master-cloud-provider
EOF
fi

mkdir -p /var/lib/logbridge
cat >/etc/origin/node/pods/logbridge.yaml <<'EOF'
apiVersion: v1
kind: Pod
metadata:
  name: logbridge
  namespace: kube-system
spec:
  containers:
  - image: {{ .Config.LogBridgeImage | quote }}
    imagePullPolicy: Always
    name: logbridge
    securityContext:
      privileged: true
    volumeMounts:
    - mountPath: /state
      name: state
    - mountPath: /cloudprovider
      name: master-cloud-provider
      readOnly: true
    - mountPath: /etc
      name: etc
      readOnly: true
    - mountPath: /var/log
      name: var-log
      readOnly: true
  hostNetwork: true
  volumes:
  - hostPath:
      path: /var/lib/logbridge
    name: state
  - hostPath:
      path: /etc/origin/cloudprovider
    name: master-cloud-provider
  - hostPath:
      path: /etc
    name: etc
  - hostPath:
      path: /var/log
    name: var-log
EOF

sed -i -re "s#( *server: ).*#\1https://$(hostname)#" /etc/origin/master/openshift-master.kubeconfig
sed -i -re "s#( *server: ).*#\1https://$(hostname)#" /etc/origin/node/node.kubeconfig
# HACK: copy node.kubeconfig to bootstrap.kubeconfig so that openshift start node used in the sync
# daemonset will not fail and set the master node labels correctly.
cp /etc/origin/node/node.kubeconfig /etc/origin/node/bootstrap.kubeconfig

# note: ${SERVICE_TYPE}-node crash loops until master is up
systemctl enable ${SERVICE_TYPE}-node.service
systemctl start ${SERVICE_TYPE}-node.service &

mkdir -p /root/.kube
cp /etc/origin/master/admin.kubeconfig /root/.kube/config
