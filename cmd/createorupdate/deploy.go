package main

import (
	"context"
	"encoding/json"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/log"
)

func deploy(cs *api.OpenShiftManagedCluster, p api.Plugin, azuredeploy []byte) error {
	var t map[string]interface{}
	err := json.Unmarshal(azuredeploy, &t)
	if err != nil {
		return err
	}

	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return err
	}

	dcli := resources.NewDeploymentsClient(cs.Properties.AzProfile.SubscriptionID)
	dcli.Authorizer = authorizer

	log.Info("creating/updating deployment")
	future, err := dcli.CreateOrUpdate(context.Background(), cs.Properties.AzProfile.ResourceGroup, "azuredeploy", resources.Deployment{
		Properties: &resources.DeploymentProperties{
			Template: t,
			Mode:     resources.Incremental,
		},
	})
	if err != nil {
		return err
	}

	log.Info("waiting for deployment")
	err = future.WaitForCompletion(context.Background(), dcli.Client)
	if err != nil {
		return err
	}

	log.Info("saving cluster state to storage account")
	return p.InitializeCluster(context.Background(), cs)
}
