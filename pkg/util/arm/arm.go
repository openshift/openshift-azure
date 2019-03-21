package arm

import (
	"sort"
	"strings"

	"github.com/openshift/openshift-azure/pkg/util/jsonpath"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
)

type Template struct {
	Schema         string        `json:"$schema,omitempty"`
	ContentVersion string        `json:"contentVersion,omitempty"`
	Parameters     struct{}      `json:"parameters,omitempty"`
	Variables      struct{}      `json:"variables,omitempty"`
	Resources      []interface{} `json:"resources,omitempty"`
	Outputs        struct{}      `json:"outputs,omitempty"`
}

// FixupAPIVersions inserts an apiVersion field into the ARM template for each
// resource (the field is missing from the internal Azure type).  The versions
// referenced here must be kept in lockstep with the imports above.
func FixupAPIVersions(template map[string]interface{}) {
	for _, resource := range jsonpath.MustCompile("$.resources.*").Get(template) {
		typ := jsonpath.MustCompile("$.type").MustGetString(resource)
		var apiVersion string
		switch typ {
		case "Microsoft.Compute/virtualMachines",
			"Microsoft.Compute/virtualMachines/extensions",
			"Microsoft.Compute/virtualMachineScaleSets":
			apiVersion = "2018-10-01"
		case "Microsoft.Network/loadBalancers",
			"Microsoft.Network/networkSecurityGroups",
			"Microsoft.Network/networkInterfaces",
			"Microsoft.Network/publicIPAddresses",
			"Microsoft.Network/virtualNetworks":
			apiVersion = "2018-07-01"
		case "Microsoft.Storage/storageAccounts":
			apiVersion = "2018-02-01"
		default:
			panic("unimplemented: " + typ)
		}
		jsonpath.MustCompile("$.apiVersion").Set(resource, apiVersion)
	}
}

// FixupDepends inserts a dependsOn field into the ARM template for each
// resource that needs it (the field is missing from the internal Azure type).
func FixupDepends(subscriptionID, resourceGroup string, template map[string]interface{}) {
	myResources := map[string]struct{}{}
	for _, resource := range jsonpath.MustCompile("$.resources.*").Get(template) {
		typ := jsonpath.MustCompile("$.type").MustGetString(resource)
		name := jsonpath.MustCompile("$.name").MustGetString(resource)

		myResources[resourceid.ResourceID(subscriptionID, resourceGroup, typ, name)] = struct{}{}
	}

	var recurse func(myResourceID string, i interface{}, dependsMap map[string]struct{})

	// walk the data structure collecting "id" fields whose values look like
	// Azure resource IDs.  Trim sub-resources from IDs.  Ignore IDs that are
	// self-referent
	recurse = func(myResourceID string, i interface{}, dependsMap map[string]struct{}) {
		switch i := i.(type) {
		case map[string]interface{}:
			if id, ok := i["id"]; ok {
				if id, ok := id.(string); ok {
					parts := strings.Split(id, "/")
					if len(parts) > 9 {
						parts = parts[:9]
					}
					if len(parts) == 9 {
						id = strings.Join(parts, "/")
						if id != myResourceID {
							dependsMap[id] = struct{}{}
						}
					}
				}
			}
			for _, v := range i {
				recurse(myResourceID, v, dependsMap)
			}
		case []interface{}:
			for _, v := range i {
				recurse(myResourceID, v, dependsMap)
			}
		}
	}

	for _, resource := range jsonpath.MustCompile("$.resources.*").Get(template) {
		typ := jsonpath.MustCompile("$.type").MustGetString(resource)
		name := jsonpath.MustCompile("$.name").MustGetString(resource)

		dependsMap := map[string]struct{}{}

		// if we're a child resource, depend on our parent
		if strings.Count(typ, "/") == 2 {
			id := resourceid.ResourceID(subscriptionID, resourceGroup, typ[:strings.LastIndexByte(typ, '/')], name[:strings.LastIndexByte(name, '/')])
			dependsMap[id] = struct{}{}
		}

		recurse(resourceid.ResourceID(subscriptionID, resourceGroup, typ, name), resource, dependsMap)

		depends := make([]string, 0, len(dependsMap))
		for k := range dependsMap {
			if _, found := myResources[k]; found {
				depends = append(depends, k)
			}
		}

		if len(depends) > 0 {
			sort.Strings(depends)

			jsonpath.MustCompile("$.dependsOn").Set(resource, depends)
		}
	}
}