package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"

	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin"
)

func pluginToDevVersion(pluginVersion string, previous int) (string, error) {
	var major, minor int
	fmt.Sscanf(pluginVersion, "v%d.%d", &major, &minor)
	if minor > 0 {
		minor -= previous
	}
	if minor == 0 {
		return fmt.Sprintf("v%d", major), nil
	}
	if minor < 0 {
		return "", fmt.Errorf("no more minor versions to try")
	}
	return fmt.Sprintf("v%d%d", major, minor), nil
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

	count := 0
	for err == nil {
		dirVer, err := pluginToDevVersion(template.PluginVersion, count)
		if err != nil {
			panic(err)
		}
		if _, err := os.Stat("pkg/sync/" + dirVer); !os.IsNotExist(err) {
			fmt.Printf(dirVer)
			return
		}
		count++
	}
}
