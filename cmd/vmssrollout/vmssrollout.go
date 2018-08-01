package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/ghodss/yaml"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/plugin"
	"github.com/openshift/openshift-azure/pkg/upgrade"
)

var (
	// vmss parameters
	subscriptionID = flag.String("subscription", "", "Azure subscription ID")
	resourceGroup  = flag.String("resource-group", "", "Azure resource group")
	name           = flag.String("name", "", "Name of the Virtual Machine Scale Set to upgrade")

	// upgrade parameters
	newCsPath = flag.String("new-config", "", "Path to the new container service config")
	oldCsPath = flag.String("old-config", "", "Path to the old container service config")
	inPlace   = flag.Bool("in-place", false, "Perform an in-place upgrade")

	// Kubernetes-specific
	drain = flag.Bool("drain", false, "Perform a Kubernetes node drain")
	role  = flag.String("role", "", "The role of the Openshift node backed by the scale set")
)

func validate() error {
	if *role == "" {
		return errors.New("node role is required")
	}
	if *role != "master" && *role != "infra" && *role != "compute" {
		return fmt.Errorf("invalid role: %s, supported roles: master, infra, compute", *role)
	}
	return nil
}

func getUpgrader(newPath, oldPath string) (*upgrade.VMSSUpgrader, error) {
	newCsBytes, err := readFile(newPath)
	if err != nil {
		return nil, err
	}
	oldCsBytes, err := readFile(oldPath)
	if err != nil {
		return nil, err
	}

	var newCs, oldCs *api.OpenShiftManagedCluster
	if err := yaml.Unmarshal(newCsBytes, &newCs); err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(oldCsBytes, &oldCs); err != nil {
		return nil, err
	}

	var imageRef *compute.ImageReference
	if newCs.Config.ImageSKU != oldCs.Config.ImageSKU ||
		newCs.Config.ImageVersion != oldCs.Config.ImageVersion {
		imageRef = &compute.ImageReference{
			Sku:     &newCs.Config.ImageSKU,
			Version: &newCs.Config.ImageVersion,
		}
	}

	if newCs.Config.ImageResourceGroup != oldCs.Config.ImageResourceGroup ||
		newCs.Config.ImageResourceName != oldCs.Config.ImageResourceName {
		id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/images/%s",
			*subscriptionID, newCs.Config.ImageResourceGroup, newCs.Config.ImageResourceName)
		imageRef = &compute.ImageReference{
			ID: &id,
		}
	}

	count, err := getCount(*role, newCs)
	if err != nil {
		return nil, err
	}

	return &upgrade.VMSSUpgrader{
		SubscriptionID: *subscriptionID,
		ResourceGroup:  *resourceGroup,
		Name:           *name,

		ImageRef: imageRef,
		Count:    int64(count),
		Drain:    *drain,
		InPlace:  *inPlace,
	}, nil
}

func readFile(path string) ([]byte, error) {
	if len(path) == 0 {
		return nil, errors.New("a path must be specified")
	}
	return ioutil.ReadFile(path)
}

func getCount(role string, cs *api.ContainerService) (int, error) {
	for _, agentProfiles := range cs.Properties.AgentPoolProfiles {
		if string(agentProfiles.Role) == role {
			return agentProfiles.Count, nil
		}
	}
	return 0, fmt.Errorf("agentPoolProfile with role %q not found in config", role)
}

func main() {
	flag.Parse()
	if err := validate(); err != nil {
		log.Fatal(err)
	}

	ssu, err := getUpgrader(*newCsPath, *oldCsPath)
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := upgrade.NewClientset()
	if err != nil {
		log.Fatal(err)
	}

	var p api.Upgrade
	switch *role {
	case "master":
		p = &plugin.MasterUpgrade{Clientset: clientset}
	case "infra":
		p = &plugin.InfraUpgrade{Clientset: clientset}
	case "compute":
		p = &plugin.ComputeUpgrade{Clientset: clientset}
	}
	ssu.Plugin = p

	if err := ssu.Upgrade(); err != nil {
		log.Fatal(err)
	}
}
