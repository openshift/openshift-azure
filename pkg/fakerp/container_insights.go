package fakerp

//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -prefix data data/...
//go:generate gofmt -s -l -w bindata.go

import (
	"context"
	"encoding/json"

	azresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/resources"
	utiltemplate "github.com/openshift/openshift-azure/pkg/util/template"
	"github.com/openshift/openshift-azure/pkg/util/workspace"
)

func createOrUpdateContainerInsights(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster) error {
	// https://raw.githubusercontent.com/microsoft/OMS-docker/ci_feature_prod/docs/templates/azuremonitor-containerSolution.json
	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
	if err != nil {
		return err
	}

	tmpl, err := Asset("azuremonitor-containerSolution.json")
	if err != nil {
		return err
	}

	// note use the resource group of the workspace not of the cluster
	_, rg, workspaceName, err := workspace.ParseLogAnalyticsResourceID(cs.Properties.MonitorProfile.WorkspaceResourceID)
	if err != nil {
		return err
	}

	b, err := utiltemplate.Template("azuremonitor-containerSolution.json", string(tmpl), nil, map[string]interface{}{
		"Location":            cs.Location,
		"SubscriptionID":      cs.Properties.AzProfile.SubscriptionID,
		"ResourceGroup":       rg,
		"WorkspaceResourceID": cs.Properties.MonitorProfile.WorkspaceResourceID,
		"WorkspaceName":       workspaceName,
	})
	if err != nil {
		return err
	}
	var azuretemplate map[string]interface{}
	err = json.Unmarshal(b, &azuretemplate)
	if err != nil {
		return err
	}

	log.Info("applying arm template deployment")
	deployments := resources.NewDeploymentsClient(ctx, log, cs.Properties.AzProfile.SubscriptionID, authorizer)
	future, err := deployments.CreateOrUpdate(ctx, rg, "azuremonitor-containerSolution", azresources.Deployment{
		Properties: &azresources.DeploymentProperties{
			Template: azuretemplate,
			Mode:     azresources.Incremental,
		},
	})
	if err != nil {
		return err
	}

	log.Info("waiting for arm template deployment to complete")
	return future.WaitForCompletionRef(ctx, deployments.Client())
}
