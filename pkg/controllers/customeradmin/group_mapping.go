package customeradmin

import (
	"context"
	"reflect"
	"sort"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/openshift/api/user/v1"
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
		g.Users = append(g.Users, *user.UserPrincipalName)
	}
	sort.Strings(g.Users)
	return g, !reflect.DeepEqual(kubeGroup, g)
}

// https://docs.microsoft.com/en-us/graph/api/group-list
// To list AAD groups, this code needs to have the clientID
// of an application with the following permissions:
// API: Microsoft Graph
//   Application permissions:
//      Read all groups
func newAADGroupsClient(config api.AADIdentityProvider) (*graphrbac.GroupsClient, error) {
	authorizer, err := azureclient.NewAuthorizer(config.ClientID, config.Secret, config.TenantID, azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return nil, err
	}

	gc := graphrbac.NewGroupsClient(config.TenantID)
	gc.Authorizer = authorizer
	return &gc, nil
}

func getAADGroupMembers(gc *graphrbac.GroupsClient, groupID string) ([]graphrbac.User, error) {
	users, err := gc.GetGroupMembers(context.Background(), groupID)
	if err != nil {
		return nil, err
	}
	members := []graphrbac.User{}
	for users.NotDone() {
		for _, bdo := range users.Values() {
			user, isUser := bdo.AsUser()
			if isUser {
				members = append(members, *user)
			}
		}

		if err = users.Next(); err != nil {
			return nil, err
		}
	}
	return members, nil
}
