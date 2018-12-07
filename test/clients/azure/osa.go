package azure

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	"github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/util/log"
)

// OSAResourceGroup returns the name of the resource group holding the OSA
// cluster resources
func (cli *Client) OSAResourceGroup(ctx context.Context, resourcegroup, name, location string) (string, error) {
	appName := strings.Join([]string{"OS", resourcegroup, name, location}, "_")

	app, err := cli.Applications.Get(ctx, appName, appName)
	if err != nil {
		return "", err
	}
	if app.ApplicationProperties == nil {
		return "", fmt.Errorf("managed application %q not found", appName)
	}

	// can't use azure.ParseResourceID here because rgid is of the short form
	// /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}
	rgid := *app.ApplicationProperties.ManagedResourceGroupID
	return rgid[strings.LastIndexByte(rgid, '/')+1:], nil
}

// UpdateOSACluster updates an OpenshiftManagedCluster by providing both the
// currently applied external manifest and an updated internal manifest
// NOTE: This method will go away as soon as the admin API is ready. Until then,
// only the KeyRotation or other e2e tests needing admin functionality should
// call this method.
func (cli *Client) UpdateOSACluster(ctx context.Context, external *v20180930preview.OpenShiftManagedCluster, config *api.PluginConfig) (*v20180930preview.OpenShiftManagedCluster, error) {
	// Remove the provisioning state before updating, if set
	external.Properties.ProvisioningState = nil

	// create logger
	logrus.SetLevel(log.SanitizeLogLevel("Debug"))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logger := logrus.NewEntry(logrus.StandardLogger())

	var oc *v20180930preview.OpenShiftManagedCluster
	var err error
	// simulate the API call to the RP
	if oc, err = fakerp.CreateOrUpdate(ctx, external, logger, config); err != nil {
		return nil, err
	}
	return oc, nil
}
