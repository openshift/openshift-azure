package assets

import (
	"github.com/openshift/installer/pkg/asset"

	"github.com/openshift/openshift-azure/pkg/assets/manifests"
)

var (
	// AroManifests are the manifests targeted assets.
	AroManifests = []asset.WritableAsset{
		&manifests.AroOperator{},
	}
)
