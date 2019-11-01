package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ghodss/yaml"
)

type VersionConfig struct {
	ImageVersion string            `json:"imageVersion,omitempty"`
	Images       map[string]string `json:"images,omitempty"`
}

type simpleConfig struct {
	Versions map[string]VersionConfig `json:"versions,omitempty"`
}

func validate(template *simpleConfig) error {
	for pluginVersion, config := range template.Versions {
		vmVersion := config.ImageVersion
		vmOcpVersionSplit := strings.Split(vmVersion, ".")
		if len(vmOcpVersionSplit) < 2 {
			return fmt.Errorf("%s] ImageVersion %s has no '.'", pluginVersion, vmVersion)
		}
		vmOcpVersion := vmOcpVersionSplit[1]
		for image, urlWithTag := range config.Images {
			// image: url
			// alertManager: registry.access.redhat.com/openshift3/prometheus-alertmanager:v3.11.129
			urlSplit := strings.Split(urlWithTag, ":")
			if len(urlSplit) != 2 {
				return fmt.Errorf("%s] %s %s has no tag", pluginVersion, image, urlWithTag)
			}
			url, tag := urlSplit[0], urlSplit[1]
			if strings.Contains(url, "registry.access.redhat.com/openshift3") {
				tagSplit := strings.Split(tag, ".")
				if len(tagSplit) != 3 {
					return fmt.Errorf("%s] tag %s is not in the form v3.11.<minor>", pluginVersion, tag)
				}
				if tagSplit[2] != vmOcpVersion {
					return fmt.Errorf("%s] VM version %s and container tag %s do not match", pluginVersion, vmVersion, tag)
				}
			}
		}
	}
	return nil
}

// Return the latest dev version. This is the way we name the versioned directories.
// To do this we read the pluginversion to get the current version and convert it.
func main() {
	b, err := ioutil.ReadFile("pluginconfig/pluginconfig-311.yaml")
	if err != nil {
		panic(err)
	}

	var template *simpleConfig
	err = yaml.Unmarshal(b, &template)
	if err != nil {
		panic(err)
	}
	err = validate(template)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
