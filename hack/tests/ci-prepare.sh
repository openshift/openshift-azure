#!/bin/bash -e

delete() {
  realrp="$1"

  if [[ -n "$NO_DELETE" ]]; then
      return
  fi

  # only delete for fakerp
  if [[ -z "$realrp" ]]; then
    make delete
  fi

}

if [[ ! -e /var/run/secrets/kubernetes.io ]]; then
    echo "Runnin not in CI"
    return
fi

pullnumber="$(python -c 'import json, os; o=json.loads(os.environ["JOB_SPEC"]); print "%s-" % o["refs"]["pulls"][0]["number"]' 2>/dev/null || true)"
export RESOURCEGROUP="ci-$pullnumber$(basename "$0" .sh)-$(cat /dev/urandom | tr -dc 'a-z' | fold -w 6 | head -n 1)"

echo "RESOURCEGROUP is $RESOURCEGROUP"
echo

make secrets

. ./secrets/secret
export AZURE_CLIENT_ID="$AZURE_CI_CLIENT_ID"
export AZURE_CLIENT_SECRET="$AZURE_CI_CLIENT_SECRET"
# currently image is only in eastus
export AZURE_REGIONS=eastus
export OPENSHIFT_INSTALL_OS_IMAGE_OVERRIDE="/resourceGroups/rhcosimages/providers/Microsoft.Compute/images/rhcos-410.8.20190504.0-azure.vhd"

az login --service-principal -u ${AZURE_CLIENT_ID} -p ${AZURE_CLIENT_SECRET} --tenant ${AZURE_TENANT_ID} >/dev/null
