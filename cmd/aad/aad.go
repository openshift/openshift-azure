package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/kelseyhightower/envconfig"

	"github.com/openshift/openshift-azure/pkg/fakerp"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

type aadCmd struct {
	action       func(ctx context.Context) (*aadOut, error)
	client       azureclient.RBACApplicationsClient
	nameOrID     string
	callbackURL  string
	Username     string `envconfig:"AZURE_USERNAME" required:"true"`
	Password     string `envconfig:"AZURE_PASSWORD" required:"true"`
	TenantID     string `envconfig:"AZURE_TENANT_ID" required:"true"`
	ClientID     string `envconfig:"AZURE_CLIENT_ID" required:"true"`
	ClientSecret string `envconfig:"AZURE_CLIENT_SECRET" required:"true"`
}

type aadOut struct {
	environment map[string]string
	notes       string
}

func usage(program string) {
	const usage string = `usage:

%[1]s app-create name callbackurl
%[1]s app-delete appId
%[1]s app-update appId callbackurl

Examples:
%[1]s app-create test-app https://openshift.test.osadev.cloud/oauth2callback/Azure%%20AD
%[1]s app-delete 76a604b8-0896-4ab7-9ef4-xxxxxxxxxx
%[1]s app-update 76a604b8-0896-4ab7-9ef4-xxxxxxxxxx https://openshift.newtest.osadev.cloud/oauth2callback/Azure%%20AD
`
	fmt.Printf(usage, program)
	os.Exit(1)
}

func newAadClientFromArgsAndEnv(args []string) (*aadCmd, error) {
	var a aadCmd
	if err := envconfig.Process("", &a); err != nil {
		return nil, err
	}
	var cmd string
	if len(args) >= 1 {
		cmd = args[0]
	}
	if len(args) >= 2 {
		a.nameOrID = args[1]
	}
	switch cmd {
	case "app-create":
		if len(args) < 3 {
			return nil, fmt.Errorf("%s needs a Name and callbackURL", cmd)
		}
		a.action = a.appCreate
		a.callbackURL = args[2]
	case "app-delete":
		if len(args) < 2 {
			return nil, fmt.Errorf("%s needs an ID", cmd)
		}
		a.action = a.appDelete
	case "app-update":
		if len(args) < 3 {
			return nil, fmt.Errorf("%s needs an ID and callbackURL", cmd)
		}
		a.action = a.appUpdate
		a.callbackURL = args[2]
	default:
		return nil, fmt.Errorf("unknown action \"%s\"", cmd)
	}
	authorizer, err := azureclient.NewAadAuthorizer(a.Username, a.Password, a.TenantID)
	if err != nil {
		return nil, err
	}
	a.client = azureclient.NewRBACApplicationsClient(a.TenantID, authorizer, []string{"en-us"})
	return &a, nil
}

func (a *aadCmd) appCreate(ctx context.Context) (*aadOut, error) {
	newPc := fakerp.NewAADPasswordCredential()
	parameters := graphrbac.ApplicationCreateParameters{
		DisplayName:             &a.nameOrID,
		Homepage:                &a.callbackURL,
		ReplyUrls:               &[]string{a.callbackURL},
		IdentifierUris:          &[]string{a.callbackURL},
		AvailableToOtherTenants: to.BoolPtr(false),
		PasswordCredentials:     &newPc,
		RequiredResourceAccess: &[]graphrbac.RequiredResourceAccess{
			{
				ResourceAppID: to.StringPtr("00000003-0000-0000-c000-000000000000"),
				ResourceAccess: &[]graphrbac.ResourceAccess{
					{
						ID:   to.StringPtr("7ab1d382-f21e-4acd-a863-ba3e13f7da61"),
						Type: to.StringPtr("Role"),
					},
					{
						ID:   to.StringPtr("5f8c59db-677d-491f-a6b8-5f174b11ec1d"),
						Type: to.StringPtr("Scope"),
					},
					{
						ID:   to.StringPtr("5b567255-7703-4780-807c-7be8301ae99b"),
						Type: to.StringPtr("Role"),
					},
					{
						ID:   to.StringPtr("37f7f235-527c-4136-accd-4a02d197296e"),
						Type: to.StringPtr("Scope"),
					},
				},
			},
			{
				ResourceAppID: to.StringPtr("00000002-0000-0000-c000-000000000000"),
				ResourceAccess: &[]graphrbac.ResourceAccess{
					{
						ID:   to.StringPtr("311a71cc-e848-46a1-bdf8-97ff7156d8e6"),
						Type: to.StringPtr("Role"),
					},
				},
			},
		},
	}
	app, err := a.client.Create(ctx, parameters)
	if err != nil {
		return nil, err
	}
	return &aadOut{
		environment: map[string]string{
			"AZURE_AAD_CLIENT_ID":     *app.AppID,
			"AZURE_AAD_CLIENT_SECRET": *newPc[0].Value,
		},
		notes: fmt.Sprintf(`Note: For the application to work, an Organization Administrator needs to grant permissions first.
		Once it is approved, it can be reused for other clusters using app-update functionality

		To use this AAD application with OpenShift cluster value below must be present in your env before creating the cluster
		export AZURE_AAD_CLIENT_ID=%s
	`, *app.AppID),
	}, nil
}

func (a *aadCmd) appDelete(ctx context.Context) (*aadOut, error) {
	_, err := a.client.Delete(ctx, a.nameOrID)
	return nil, err
}

func (a *aadCmd) appUpdate(ctx context.Context) (*aadOut, error) {
	azureAadClientSecret, err := fakerp.UpdateAADAppSecret(ctx, a.client, a.nameOrID, a.callbackURL)
	if err != nil {
		return nil, err
	}
	out := aadOut{
		environment: map[string]string{
			"AZURE_AAD_CLIENT_ID":     a.nameOrID,
			"AZURE_AAD_CLIENT_SECRET": azureAadClientSecret,
		},
	}
	return &out, nil
}

func main() {
	a, err := newAadClientFromArgsAndEnv(flag.Args())
	if err != nil {
		fmt.Println()
		fmt.Println("Error parsing args and environment: ", err)
		fmt.Println()
		usage(os.Args[0])
	}
	out, err := a.action(context.Background())
	if err != nil {
		fmt.Printf("%s] %v", os.Args[1], err)
		os.Exit(1)
	}
	if out != nil {
		outArr := []string{}
		for name, val := range out.environment {
			outArr = append(outArr, fmt.Sprintf("%s=%s", name, val))
		}
		fmt.Println(strings.Join(outArr, "\n"))
		fmt.Println(out.notes)
	}
}
