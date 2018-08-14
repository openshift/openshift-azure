package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/equality"

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
	csPath          = flag.String("config", "", "Path to the latest container service config")
	oldCsPath       = flag.String("old-config", "", "Path to the old container service config")
	templateFile    = flag.String("template-file", "", "Path to the latest ARM template")
	oldTemplateFile = flag.String("old-template-file", "", "Path to the old ARM template")
	inPlace         = flag.Bool("in-place", false, "Perform an in-place upgrade")

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
		log.Print("Image updated")
	}

	if newCs.Config.ImageResourceGroup != oldCs.Config.ImageResourceGroup ||
		newCs.Config.ImageResourceName != oldCs.Config.ImageResourceName {
		id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/images/%s",
			*subscriptionID, newCs.Config.ImageResourceGroup, newCs.Config.ImageResourceName)
		imageRef = &compute.ImageReference{
			ID: &id,
		}
		log.Print("Image updated")
	}

	count, err := getCount(*role, newCs)
	if err != nil {
		return nil, err
	}

	script, err := getScript(*templateFile, *oldTemplateFile)
	if err != nil {
		return nil, err
	}

	return &upgrade.VMSSUpgrader{
		SubscriptionID: *subscriptionID,
		ResourceGroup:  *resourceGroup,
		Name:           *name,

		Script:   script,
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

func getScript(newPath, oldPath string) (map[string]interface{}, error) {
	newScript, err := readScriptFromTemplate(newPath)
	if err != nil {
		return nil, err
	}
	oldScript, err := readScriptFromTemplate(oldPath)
	if err != nil {
		return nil, err
	}
	if equality.Semantic.DeepEqual(newScript, oldScript) {
		return nil, nil
	}
	log.Print("Script updated")
	return newScript, nil
}

func readScriptFromTemplate(path string) (map[string]interface{}, error) {
	templateBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	templateJSON := make(map[string]interface{})
	if err := json.Unmarshal(templateBytes, &templateJSON); err != nil {
		return nil, err
	}

	resourceSlice := templateJSON["resources"].([]interface{})
	var vmss *compute.VirtualMachineScaleSet
	var result interface{}
	for _, r := range resourceSlice {
		res := r.(map[string]interface{})
		for k, v := range res {
			val, ok := v.(string)
			if k == "name" && ok && val == *name {
				result = r
				break
			}
		}
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	vmss = &compute.VirtualMachineScaleSet{}
	if err := vmss.UnmarshalJSON(data); err != nil {
		return nil, err
	}

	extensions := *vmss.VirtualMachineScaleSetProperties.VirtualMachineProfile.ExtensionProfile.Extensions
	for _, ext := range extensions {
		if ext.Name != nil && *ext.Name == "cse" {
			return ext.ProtectedSettings.(map[string]interface{}), nil
		}
	}

	return nil, nil
}

func getCount(role string, cs *api.OpenShiftManagedCluster) (int, error) {
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

	ssu, err := getUpgrader(*csPath, *oldCsPath)
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
