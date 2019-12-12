package vmimage

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-10-01/compute"
	azresources "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/arm"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/resources"
	"github.com/openshift/openshift-azure/pkg/util/resourceid"
	"github.com/openshift/openshift-azure/pkg/util/template"
	"github.com/openshift/openshift-azure/pkg/util/tls"
)

//go:generate ../../hack/build-archive.sh
//go:generate go get github.com/go-bindata/go-bindata/go-bindata
//go:generate go-bindata -nometadata -pkg $GOPACKAGE -prefix data data
//go:generate gofmt -s -l -w bindata.go

// Builder is the VM image configuration struct
type Builder struct {
	GitCommit                  string
	Log                        *logrus.Entry
	Deployments                resources.DeploymentsClient
	Groups                     resources.GroupsClient
	SubscriptionID             string
	Location                   string
	BuildResourceGroup         string
	PreserveBuildResourceGroup bool
	DomainNameLabel            string
	Image                      string
	ImageResourceGroup         string
	ImageStorageAccount        string
	ImageContainer             string
	ImageSku                   string
	ImageVersion               string
	SSHKey                     *rsa.PrivateKey
	ClientKey                  *rsa.PrivateKey
	ClientCert                 *x509.Certificate

	Validate bool
}

func (builder *Builder) generateTemplate() (map[string]interface{}, error) {
	var script []byte
	var err error
	if !builder.Validate {
		script, err = template.Template("script.sh", string(MustAsset("script.sh")), nil, map[string]interface{}{
			"Archive":      MustAsset("archive.tgz"),
			"Builder":      builder,
			"ClientID":     os.Getenv("AZURE_CLIENT_ID"),
			"ClientSecret": os.Getenv("AZURE_CLIENT_SECRET"),
			"TenantID":     os.Getenv("AZURE_TENANT_ID"),
		})
		if err != nil {
			return nil, err
		}
	} else {
		script, err = template.Template("validate.sh", string(MustAsset("validate.sh")), nil, map[string]interface{}{
			"Archive": MustAsset("archive.tgz"),
			"Builder": builder,
		})
		if err != nil {
			return nil, err
		}
	}

	cse, err := cse(builder.Location, script)
	if err != nil {
		return nil, err
	}

	sshPublicKey, err := tls.SSHPublicKeyAsString(&builder.SSHKey.PublicKey)
	if err != nil {
		return nil, err
	}

	imageReference := builder.vmImageReference()
	var vmPlan *compute.Plan
	if builder.hasMarketplaceVMImageRef() {
		vmPlan = &compute.Plan{
			Name:      imageReference.Sku,
			Publisher: imageReference.Publisher,
			Product:   imageReference.Offer,
		}
	}

	t := arm.Template{
		Schema:         "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		ContentVersion: "1.0.0.0",
		Resources: []*arm.Resource{
			vnet(builder.Location),
			ip(builder.BuildResourceGroup, builder.Location, builder.DomainNameLabel),
			nsg(builder.Location),
			nic(builder.SubscriptionID, builder.BuildResourceGroup, builder.Location),
			vm(builder.SubscriptionID, builder.BuildResourceGroup, builder.Location, sshPublicKey, vmPlan, imageReference),
			cse,
		},
	}

	b, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	var template map[string]interface{}
	err = json.Unmarshal(b, &template)
	if err != nil {
		return nil, err
	}

	err = arm.FixupAPIVersions(template, versionMap)
	if err != nil {
		return nil, err
	}

	arm.FixupDepends(builder.SubscriptionID, builder.BuildResourceGroup, template, nil)

	return template, nil
}

func (builder *Builder) vmImageReference() *compute.ImageReference {
	if !builder.Validate {
		// Building a new image in a VM
		return &compute.ImageReference{
			Publisher: to.StringPtr("RedHat"),
			Offer:     to.StringPtr("RHEL"),
			Sku:       to.StringPtr("7-RAW"),
			Version:   to.StringPtr("latest"),
		}
	}

	// Validating an existing marketplace image
	if builder.hasMarketplaceVMImageRef() {
		return &compute.ImageReference{
			Publisher: to.StringPtr("redhat"),
			Offer:     to.StringPtr("osa"),
			Sku:       to.StringPtr(builder.ImageSku),
			Version:   to.StringPtr(builder.ImageVersion),
		}
	}

	// Validating an existing custom image in one of our resource groups
	return &compute.ImageReference{
		ID: to.StringPtr(resourceid.ResourceID(
			builder.SubscriptionID,
			builder.ImageResourceGroup,
			"Microsoft.Compute/images",
			builder.Image,
		)),
	}
}

func (builder *Builder) hasMarketplaceVMImageRef() bool {
	return builder.ImageSku != "" && builder.ImageVersion != ""
}

func (builder *Builder) hasCustomVMIMageRef() bool {
	return builder.Image != "" &&
		builder.ImageResourceGroup != "" &&
		builder.ImageStorageAccount != "" &&
		builder.ImageContainer != ""

}

// ValidateFields makes sure that the builder struct has correct values in fields
func (builder *Builder) ValidateFields() error {
	vmImageRefFields := []string{"ImageSku", "ImageVersion"}
	customVMIMageRefFields := []string{"Image", "ImageResourceGroup", "ImageStorageAccount", "ImageContainer"}

	if !builder.hasMarketplaceVMImageRef() && !builder.hasCustomVMIMageRef() {
		return fmt.Errorf(
			"missing fields: you must provide values for either %s fields or %s fields",
			strings.Join(vmImageRefFields, ", "),
			strings.Join(customVMIMageRefFields, ", "),
		)
	}

	if builder.hasMarketplaceVMImageRef() && builder.hasCustomVMIMageRef() {
		return fmt.Errorf(
			"confilicting fields: you must provide values for either %s fields or %s fields",
			strings.Join(vmImageRefFields, ", "),
			strings.Join(customVMIMageRefFields, ", "),
		)
	}

	return nil
}

// Run is the main entry point
func (builder *Builder) Run(ctx context.Context) error {
	if builder.hasCustomVMIMageRef() {
		builder.Log.Debugf("using %s image in %s resource group", builder.Image, builder.ImageResourceGroup)
	} else {
		builder.Log.Debugf("using: redhat osa image (SKU: %s; Version: %s)", builder.ImageSku, builder.ImageVersion)
	}

	template, err := builder.generateTemplate()
	if err != nil {
		return err
	}

	defer func() {
		if !builder.PreserveBuildResourceGroup {
			builder.Log.Infof("PreserveBuildResourceGroup not set, deleting build resource group")
			builder.Groups.Delete(ctx, builder.BuildResourceGroup)
		}
	}()

	builder.Log.Infof("creating resource group %s", builder.BuildResourceGroup)
	_, err = builder.Groups.CreateOrUpdate(ctx, builder.BuildResourceGroup, azresources.Group{
		Location: to.StringPtr(builder.Location),
		Tags: map[string]*string{
			"now": to.StringPtr(fmt.Sprintf("%d", time.Now().Unix())),
			"ttl": to.StringPtr("6h"),
		},
	})
	if err != nil {
		return err
	}

	builder.Log.Infof("deploying template, ssh to VM if needed via:")
	builder.Log.Infof("  ssh -i id_rsa cloud-user@%s.%s.cloudapp.azure.com", builder.DomainNameLabel, builder.Location)
	future, err := builder.Deployments.CreateOrUpdate(ctx, builder.BuildResourceGroup, "azuredeploy", azresources.Deployment{
		Properties: &azresources.DeploymentProperties{
			Template: template,
			Mode:     azresources.Incremental,
		},
	})
	if err != nil {
		return err
	}

	go builder.ssh()

	cli := builder.Deployments.Client()
	cli.PollingDuration = time.Minute * 90

	builder.Log.Infof("waiting for deployment")
	err = future.WaitForCompletionRef(ctx, cli)
	if err != nil {
		return err
	}

	if builder.Validate {
		builder.Log.Infof("copy file from VM")
		err := builder.scp([]string{
			"/tmp/yum_updateinfo",
			"/tmp/yum_updateinfo_list_security",
			"/tmp/yum_check_update",
			"/tmp/scap_report.html",
		})
		if err != nil {
			return err
		}
	}

	return nil
}
