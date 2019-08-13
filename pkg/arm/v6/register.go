package arm

import (
	"github.com/openshift/openshift-azure/pkg/runtime"
)

func init() {
	runtime.AddToVersion("v6.0", New)
	runtime.AddToVersion("v6", New)
}
