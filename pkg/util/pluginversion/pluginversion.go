package pluginversion

import (
	"fmt"
)

func Parse(pluginVersion string) (major, minor int, err error) {
	_, err = fmt.Sscanf(pluginVersion, "v%d.%d", &major, &minor)
	return
}
