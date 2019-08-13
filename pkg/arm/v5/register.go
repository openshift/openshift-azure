package arm

import (
	"github.com/openshift/openshift-azure/pkg/runtime"
)

func init() {
	runtime.AddToVersion("v5.0", New)
	runtime.AddToVersion("v5", New)
}
