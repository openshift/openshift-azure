# Releasing a VM image

1. Pre-requisites:
   * `az` client installed and logged in
   * valid environment file sourced
   * secrets/client-{cert,key}.pem exist

1. Run `make vmimage`.  This:
   * creates an Azure VM, runs the image build kickstart using nested KVM
   * uploads the resulting disk image to a blob
   * creates an Image object from the blob
   * deletes the VM
   * creates a cluster using the Image object
   * runs an e2e
   * deletes the cluster
   * if successful, outputs the name of the Image object

1. Run `hack/vmimage-cloudpartner.sh $IMAGE`.  This:
   * outputs the parameters for configuration in https://cloudpartner.azure.com/

1. Manually configure and publish the image in https://cloudpartner.azure.com/

1. Update image reference in pluginconfig/pluginconfig-3.11.yaml and commit
