package main

import (
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"

	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin"
)

func pluginToDevVersion(pluginVersion string) string {
	var major, minor int
	fmt.Sscanf(pluginVersion, "v%d.%d", &major, &minor)
	if minor == 0 {
		return fmt.Sprintf("v%d", major)
	}
	return fmt.Sprintf("v%d%d", major, minor)
}

// Return the latest dev version. This is the way we name the versioned directories.
// To do this we read the pluginversion to get the current version and convert it.
func main() {
	b, err := ioutil.ReadFile("pluginconfig/pluginconfig-311.yaml")
	if err != nil {
		panic(err)
	}

	var template *pluginapi.Config
	err = yaml.Unmarshal(b, &template)
	if err != nil {
		panic(err)
	}
	fmt.Printf(pluginToDevVersion(template.PluginVersion))
}
