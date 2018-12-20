package config

import (
	"fmt"
	"strings"

	"github.com/openshift/openshift-azure/pkg/api"
)

// openshiftVersion converts a VM image version (e.g. 311.43.20181121) to an
// openshift container image version (e.g. v3.11.43)
func openShiftVersion(imageVersion string) (string, error) {
	parts := strings.Split(imageVersion, ".")
	if len(parts) != 3 || len(parts[0]) < 2 {
		return "", fmt.Errorf("invalid imageVersion %q", imageVersion)
	}

	return fmt.Sprintf("v%s.%s.%s", parts[0][:1], parts[0][1:], parts[1]), nil
}

func (g *simpleGenerator) selectNodeImage(cs *api.OpenShiftManagedCluster) {
	c := &cs.Config
	c.ImagePublisher = "redhat"
	if c.ImageOffer == "" {
		c.ImageOffer = "osa"
	}

	switch g.pluginConfig.TestConfig.DeployOS {
	case "", "rhel7":
		c.ImageSKU = "osa_" + strings.Replace(cs.Properties.OpenShiftVersion[1:], ".", "", -1)
		if c.ImageVersion == "" {
			c.ImageVersion = "311.43.20181121"
		}
	case "centos7":
		c.ImageSKU = "origin_" + strings.Replace(cs.Properties.OpenShiftVersion[1:], ".", "", -1)
		if c.ImageVersion == "" {
			c.ImageVersion = "311.0.20181109"
		}
	}
}
