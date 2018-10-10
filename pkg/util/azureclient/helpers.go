package azureclient

import (
	"encoding/json"

	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	sdk "github.com/openshift/openshift-azure/pkg/util/azureclient/osa-go-sdk/services/containerservice/mgmt/2018-09-30-preview/containerservice"
)

func SdkToExternal(in *sdk.OpenShiftManagedCluster) (out *v20180930preview.OpenShiftManagedCluster) {
	b, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(b, &out)
	if err != nil {
		panic(err)
	}

	return
}

func ExternalToSdk(in *v20180930preview.OpenShiftManagedCluster) (out *sdk.OpenShiftManagedCluster) {
	b, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(b, &out)
	if err != nil {
		panic(err)
	}

	return
}
