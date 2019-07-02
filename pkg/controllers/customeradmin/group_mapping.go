package customeradmin

import (
	"context"
	"reflect"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/azure"
	v1 "github.com/openshift/api/user/v1"
	userv1client "github.com/openshift/client-go/user/clientset/versioned/typed/user/v1"
	"github.com/sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	azgraphrbac "github.com/openshift/openshift-azure/pkg/util/azureclient/graphrbac"
)

// fromMSGraphGroup syncs the values from the given aad group into kubeGroup
// it returns a boolean "changed" that tells the call if an update is required
func fromMSGraphGroup(log *logrus.Entry, userV1 userv1client.UserV1Interface, kubeGroup *v1.Group, kubeGroupName string, msGroupMembers []graphrbac.User) (*v1.Group, bool) {
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
	// list all ocp users for reconcile process
	ocpUserList, err := userV1.Users().List(meta_v1.ListOptions{})
	if err != nil {
		log.Warnf("failed to list users %s", err)
	}

	g.Users = []string{}
	for _, user := range msGroupMembers {
		if user.UserType == graphrbac.Guest && user.Mail != nil {
			g.Users = append(g.Users, reconcileGuestUsers(log, ocpUserList.Items, user))
		} else {
			g.Users = append(g.Users, *user.UserPrincipalName)
		}
	}
	sort.Strings(g.Users)
	return g, !reflect.DeepEqual(kubeGroup, g)
}

// reconcileGuestUsers will take an external user reference from AAD
// match it with already sign-ed in users and updates required metadata
// to match external provider prefix.
// Example: foo@bar.com external guest user in AAD after sign-in
// would become live.com#foo@bar.com, where live.com# is the origin
// Function would match user and convert foo@bar.com to live.com#foo@bar.com
func reconcileGuestUsers(log *logrus.Entry, ocpUserList []v1.User, AADUser graphrbac.User) string {
	// If AAD user is External type
	// AADUser.Mail - does not contain ext reference
	// AADUser.MailNickname - does have ext reference
	if AADUser.MailNickname != nil && strings.Contains(*AADUser.MailNickname, "#EXT#") {
		for _, usr := range ocpUserList {
			// if login name exist and contains email, we gonna use it as ref
			if strings.Contains(usr.Name, *AADUser.Mail) {
				return usr.Name
			}
		}
	}
	return *AADUser.Mail
}

func newAADGroupsClient(ctx context.Context, log *logrus.Entry, config api.AADIdentityProvider) (azgraphrbac.GroupsClient, error) {
	graphauthorizer, err := azureclient.NewAuthorizer(config.ClientID, config.Secret, config.TenantID, azure.PublicCloud.GraphEndpoint)
	if err != nil {
		return nil, err
	}

	return azgraphrbac.NewGroupsClient(ctx, log, config.TenantID, graphauthorizer), nil
}

func getAADGroupMembers(gc azgraphrbac.GroupsClient, groupID string) ([]graphrbac.User, error) {
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
