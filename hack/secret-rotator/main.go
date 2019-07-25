package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	azgraphrbac "github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/Azure/go-autorest/autorest/to"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/authorization"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/graphrbac"
)

const (
	aroTeamSharedID = "8b45c9d6-0b66-44ef-93df-7a6b0c4c1635"
	aroCISharedID   = "d17e4e41-c234-4186-af55-68e3618de304"
	dateLayout      = "2006-01-02 3:4:5"
)

var (
	secretNamespace = flag.String("secret-namespace", "azure", "Secret namespace")
	secretName      = flag.String("secret-name", "cluster-secrets-azure-mj", "Secret name")
)

type credential struct {
	secretKey   string
	secret      string
	clientIDKey string
	clientID    string
}

type aadManager struct {
	log *logrus.Entry

	ac  graphrbac.ApplicationsClient
	sc  graphrbac.ServicePrincipalsClient
	rac authorization.RoleAssignmentsClient

	clientset *kubernetes.Clientset
}

func newAADManager(ctx context.Context, log *logrus.Entry) (*aadManager, error) {
	authorizer, err := azureclient.NewAuthorizerFromEnvironment("")
	if err != nil {
		return nil, err
	}

	graphauthorizer, err := azureclient.NewAuthorizerFromEnvironment(azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return nil, err
	}

	clientset, err := getClientset()
	if err != nil {
		return nil, err
	}

	return &aadManager{
		log:       log,
		ac:        graphrbac.NewApplicationsClient(ctx, log, os.Getenv("AZURE_TENANT_ID"), graphauthorizer),
		sc:        graphrbac.NewServicePrincipalsClient(ctx, log, os.Getenv("AZURE_TENANT_ID"), graphauthorizer),
		rac:       authorization.NewRoleAssignmentsClient(ctx, log, os.Getenv("AZURE_SUBSCRIPTION_ID"), authorizer),
		clientset: clientset,
	}, nil
}

// rotateSecret ensures that all ARO dev secrets are rotated
func (am *aadManager) rotateSecret(ctx context.Context, objectId string) (*credential, error) {
	results, err := am.ac.List(ctx, fmt.Sprintf("objectID eq '%s'", objectId))
	if err != nil {
		return nil, err
	}

	if len(results.Values()) != 1 {
		return nil, fmt.Errorf("found %d applications, should be 1", len(results.Values()))
	}

	app := results.Values()[0]
	passwords := []azgraphrbac.PasswordCredential{}
	for _, passwd := range *app.PasswordCredentials {
		passwords = append(passwords, passwd)
	}

	// sort newest at the end, oldest at the start
	// and remove old secrets by taking as many componets we need from the end
	if len(passwords) > 1 {
		sort.Slice(passwords, func(i, j int) bool {
			iDate, err := time.Parse(dateLayout, string(*passwords[i].CustomKeyIdentifier))
			if err != nil {
				panic(err)
			}
			jDate, err := time.Parse(dateLayout, string(*passwords[j].CustomKeyIdentifier))
			if err != nil {
				panic(err)
			}
			return iDate.Before(jDate)
		})

		// take last 2 secrets
		if len(passwords) >= 2 {
			passwords = passwords[len(passwords)-1 : len(passwords)]
		}
	}
	secret := uuid.NewV4().String()
	// application key identifier cannot be empty and can be at most 32 bytes
	timestamp := []byte(time.Now().Format(dateLayout))
	newPswd := azgraphrbac.PasswordCredential{
		CustomKeyIdentifier: &timestamp,
		Value:               &secret,
		EndDate:             &date.Time{Time: time.Now().AddDate(1, 0, 0)},
	}
	passwords = append(passwords, newPswd)

	_, err = am.ac.Patch(ctx, objectId, azgraphrbac.ApplicationUpdateParameters{
		AvailableToOtherTenants: to.BoolPtr(false),
		PasswordCredentials:     &passwords,
	})
	if err != nil {
		return nil, err
	}

	return &credential{
		secret:   secret,
		clientID: *app.AppID,
	}, nil
}

func (am *aadManager) updateSecretFile(ctx context.Context, credential *credential) error {
	secret, err := am.clientset.CoreV1().Secrets(*secretNamespace).Get(*secretName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	secretCopy := secret.DeepCopy()
	data := secretCopy.Data["secret"]

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.Contains(line, credential.secretKey) {
			lines[i] = fmt.Sprintf("export %s=%s", credential.secretKey, credential.secret)
		}
		if strings.Contains(line, credential.clientIDKey) {
			lines[i] = fmt.Sprintf("export %s=%s", credential.clientIDKey, credential.clientID)
		}
	}
	output := strings.Join(lines, "\n")

	// update the secret
	secretCopy.Data["secret"] = []byte(output)

	_, err = am.clientset.CoreV1().Secrets(*secretNamespace).Update(secretCopy)
	if err != nil {
		return err
	}

	return nil
}

func getClientset() (*kubernetes.Clientset, error) {
	// off-cluster config
	if os.Getenv("KUBECONFIG") != "" {
		// use the current context in kubeconfig
		config, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
		if err != nil {
			return nil, err
		}

		return kubernetes.NewForConfig(config)
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func main() {
	err := run()
	if err != nil {
		panic(err)
	}

}

func run() error {
	ctx := context.Background()
	am, err := newAADManager(ctx, logrus.NewEntry(logrus.StandardLogger()))
	if err != nil {
		return err
	}
	// rotate aro-team-shared secret
	// AZURE_CLIENT_ID
	// AZURE_CLIENT_SECRET
	am.log.Infof("update %s secret and application %s", "AZURE_CLIENT_ID", aroTeamSharedID)
	secret, err := am.rotateSecret(ctx, aroTeamSharedID)
	if err != nil {
		panic(err)
	}
	secret.clientIDKey = "AZURE_CLIENT_ID"
	secret.secretKey = "AZURE_CLIENT_SECRET"

	err = am.updateSecretFile(ctx, secret)
	if err != nil {
		return nil
	}

	// rotate aro-ci-team-shared secret
	// AZURE_CI_CLIENT_ID
	// AZURE_CI_CLIENT_SECRET
	am.log.Infof("update %s secret and application %s", "AZURE_CLIENT_ID", aroCISharedID)
	secret, err = am.rotateSecret(ctx, aroCISharedID)
	if err != nil {
		panic(err)
	}
	secret.clientIDKey = "AZURE_CI_CLIENT_ID"
	secret.secretKey = "AZURE_CI_CLIENT_SECRET"

	err = am.updateSecretFile(ctx, secret)
	if err != nil {
		return nil
	}

	return nil
}
