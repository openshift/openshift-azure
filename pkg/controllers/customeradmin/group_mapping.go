package customeradmin

import (
	"context"
	"reflect"
	"sort"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/azure"
	v1 "github.com/openshift/api/user/v1"
	"github.com/sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// fromMSGraphGroup syncs the values from the given aad group into kubeGroup
// it returns a boolean "changed" that tells the call if an update is required
func fromMSGraphGroup(log *logrus.Entry, kubeGroup *v1.Group, kubeGroupName string, msGroupMembers []graphrbac.User) (*v1.Group, bool) {
	var g *v1.Group
	if kubeGroup == nil {
		g = &v1.Group{
			ObjectMeta: meta_v1.ObjectMeta{
				Name: kubeGroupName,
			},
			Users: []string{},
		}
	} else {
		g = kubeGroup.DeepCopy()
	}
	g.Users = []string{}
	for _, user := range msGroupMembers {
		if user.UserType == graphrbac.Guest && user.Mail != nil {
			g.Users = append(g.Users, *user.Mail) // doesn't include #EXT#
		} else {
			g.Users = append(g.Users, *user.UserPrincipalName)
		}
	}
	sort.Strings(g.Users)
	return g, !reflect.DeepEqual(kubeGroup, g)
}

func newAADGroupsClient(ctx context.Context, config api.AADIdentityProvider) (azureclient.RBACGroupsClient, error) {
	graphauthorizer, err := azureclient.NewAuthorizer(config.ClientID, config.Secret, config.TenantID, azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return nil, err
	}

	return azureclient.NewRBACGroupsClient(ctx, config.TenantID, graphauthorizer), nil
}

func getAADGroupMembers(gc azureclient.RBACGroupsClient, groupID string) ([]graphrbac.User, error) {
	members, err := gc.GetGroupMembers(context.Background(), groupID)
	if err != nil {
		return nil, err
	}
	var users []graphrbac.User
	for _, member := range members {
		if user, ok := member.AsUser(); ok {
			users = append(users, *user)
		}
	}
	return users, nil
}
