package version

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"

	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin"
	"github.com/openshift/openshift-azure/pkg/util/pluginversion"
)

// NewCommand returns the cobra command for "dev-version".
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:  "dev-version",
		Long: "Get the development version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return start(cmd)
		},
	}
}

func nextDevVersion(pluginVersion string, previous int) (string, error) {
	major, minor, _ := pluginversion.Parse(pluginVersion)
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

func PluginToDevVersion(pluginversion string) (string, error) {
	count := 0
	var err error
	for err == nil {
		dirVer, err := nextDevVersion(pluginversion, count)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat("pkg/sync/" + dirVer); !os.IsNotExist(err) {
			return dirVer, nil
		}
		count++
	}
	return "", fmt.Errorf("impossible")
}

func getLatestDevVersion() (string, error) {
	b, err := ioutil.ReadFile("pluginconfig/pluginconfig-311.yaml")
	if err != nil {
		return "", err
	}
	var template *pluginapi.Config
	err = yaml.Unmarshal(b, &template)
	if err != nil {
		return "", err
	}
	return PluginToDevVersion(template.PluginVersion)
}

// print the latest dev version. This is the way we name the versioned directories.
// To do this we read the pluginversion to get the current version and convert it.
func start(cc *cobra.Command) error {
	ver, err := getLatestDevVersion()
	if err != nil {
		return err
	}
	fmt.Print(ver)
	return nil
}
