package arm

import (
	"github.com/openshift/openshift-azure/pkg/runtime"
)

func init() {
	runtime.AddToVersion("v7.0", New)
	runtime.AddToVersion("v7", New)
}
