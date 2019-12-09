package customeradmin

import (
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/to"
	v1 "github.com/openshift/api/user/v1"
	fakeuserv1 "github.com/openshift/client-go/user/clientset/versioned/fake"
	userv1client "github.com/openshift/client-go/user/clientset/versioned/typed/user/v1"
	"github.com/sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFromMSGraphGroup(t *testing.T) {
	// see main function for explanations
	userv1 := fakeuserv1.NewSimpleClientset(
		// case 1
		&v1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "owner@home.com",
			},
		},
		// case 2
		&v1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "live.com#user@guest.com",
			},
		},
		// case 2.b
		&v1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "live.com#hhi_admin@guest.com",
			},
		},
		// case 3
		&v1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "user@trustedGuest.com",
			},
		},
		// case 4
		&v1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "user@home.com",
			},
		},
		&v1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "bob_owner@home.com",
			},
		},
		//case 5
		&v1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "user@userprincipalname.com",
			},
		},
	).UserV1()
	tests := []struct {
		name           string
		kubeGroup      *v1.Group
		userV1         userv1client.UserV1Interface
		aadGroupID     string
		kubeGroupName  string
		msGroupMembers []graphrbac.User
		want           *v1.Group
		want1          bool
	}{
		{
			name:          "default group (no aad group)",
			want1:         true,
			kubeGroupName: osaCustomerAdmins,
			want: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{},
			},
		},
		{
			name:          "create new group",
			want1:         true,
			kubeGroupName: osaCustomerAdmins,
			msGroupMembers: []graphrbac.User{
				{
					UserPrincipalName: to.StringPtr("foo@somewhere.com"),
				},
			},
			want: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"foo@somewhere.com"},
			},
		},
		{
			name:          "no change",
			kubeGroupName: osaCustomerAdmins,
			kubeGroup: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"foo@somewhere.com"},
			},
			msGroupMembers: []graphrbac.User{
				{
					UserPrincipalName: to.StringPtr("foo@somewhere.com"),
				},
			},
			want: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"foo@somewhere.com"},
			},
		},
		{
			name:          "add to membership",
			kubeGroupName: osaCustomerAdmins,
			want1:         true,
			kubeGroup: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"foo@somewhere.com"},
			},
			msGroupMembers: []graphrbac.User{
				{
					UserPrincipalName: to.StringPtr("foo@somewhere.com"),
				},
				{
					UserPrincipalName: to.StringPtr("tim@somewhere.com"),
				},
			},
			want: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"foo@somewhere.com", "tim@somewhere.com"},
			},
		},
		{
			name:          "remove from membership",
			kubeGroupName: osaCustomerAdmins,
			want1:         true,
			kubeGroup: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"foo@somewhere.com", "tim@somewhere.com"},
			},
			msGroupMembers: []graphrbac.User{
				{
					Mail:     to.StringPtr("foo@somewhere.com"),
					UserType: graphrbac.Guest,
				},
			},
			want: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"foo@somewhere.com"},
			},
		},
		{
			name:          "owner is an admin (case 1)",
			kubeGroupName: osaCustomerAdmins,
			want1:         false,
			kubeGroup: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"owner@home.com"},
			},
			msGroupMembers: []graphrbac.User{
				{
					Mail:              nil,
					UserType:          graphrbac.Member,
					GivenName:         to.StringPtr("owner@home.com"),
					MailNickname:      to.StringPtr("owner_home.com#EXT#"),
					UserPrincipalName: to.StringPtr("owner_home.com#EXT#@home2.onmicrosoft.com"),
				},
			},
			want: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"owner@home.com"},
			},
		},
		{
			name:          "guest user with prefix (case 2)",
			kubeGroupName: osaCustomerAdmins,
			want1:         false,
			kubeGroup: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"live.com#user@guest.com"},
			},
			msGroupMembers: []graphrbac.User{
				{
					Mail:              to.StringPtr("user@guest.com"),
					UserType:          graphrbac.Guest,
					GivenName:         nil,
					MailNickname:      to.StringPtr("user_guest.com#EXT#"),
					UserPrincipalName: to.StringPtr("user_guest.com#EXT#@home2.onmicrosoft.com"),
				},
			},
			want: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"live.com#user@guest.com"},
			},
		},
		{
			name:          "guest user with prefix (case 2.b)",
			kubeGroupName: osaCustomerAdmins,
			want1:         false,
			kubeGroup: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"live.com#hhi_admin@guest.com"},
			},
			msGroupMembers: []graphrbac.User{
				{
					Mail:              nil,
					UserType:          graphrbac.Guest,
					GivenName:         nil,
					MailNickname:      to.StringPtr("hhi_admin_guest.com#EXT#"),
					UserPrincipalName: to.StringPtr("hhi_admin_guest.com#EXT#@home2.onmicrosoft.com"),
				},
			},
			want: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"live.com#hhi_admin@guest.com"},
			},
		},
		{
			name:          "guest user no with prefix (case 3)",
			kubeGroupName: osaCustomerAdmins,
			want1:         false,
			kubeGroup: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"user@trustedGuest.com"},
			},
			msGroupMembers: []graphrbac.User{
				{
					Mail:              to.StringPtr("user@trustedGuest.com"),
					UserType:          graphrbac.Guest,
					GivenName:         nil,
					MailNickname:      to.StringPtr("user_trustedGuest.com#EXT#"),
					UserPrincipalName: to.StringPtr("user_trustedGuest.com#EXT#@home2.onmicrosoft.com"),
				},
			},
			want: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"user@trustedGuest.com"},
			},
		},
		{
			name:          "guest user with Mail field matching (case 4)",
			kubeGroupName: osaCustomerAdmins,
			want1:         false,
			kubeGroup: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"user@home.com"},
			},
			msGroupMembers: []graphrbac.User{
				{
					Mail:              to.StringPtr("user@home.com"),
					UserType:          graphrbac.Guest,
					GivenName:         nil,
					MailNickname:      to.StringPtr("user_home.com#EXT#"),
					UserPrincipalName: to.StringPtr("user_home.com#EXT#@home2.onmicrosoft.com"),
				},
			},
			want: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"user@home.com"},
			},
		},
		{
			name:          "email with underscore",
			kubeGroupName: osaCustomerAdmins,
			want1:         false,
			kubeGroup: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"bob_owner@home.com"},
			},
			msGroupMembers: []graphrbac.User{
				{
					Mail:              to.StringPtr("bob_owner@home.com"),
					UserType:          graphrbac.Guest,
					GivenName:         nil,
					MailNickname:      to.StringPtr("bob_owner_home.com#EXT#"),
					UserPrincipalName: to.StringPtr("bob_owner_home.com#EXT#@home2.onmicrosoft.com"),
				},
			},
			want: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"bob_owner@home.com"},
			},
		},
		{
			name:          "leaver user",
			kubeGroupName: osaCustomerAdmins,
			want1:         true,
			kubeGroup: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"user@home.com", "leaver@example.com"},
			},
			msGroupMembers: []graphrbac.User{
				{
					Mail:              to.StringPtr("user@home.com"),
					UserType:          graphrbac.Guest,
					GivenName:         nil,
					MailNickname:      to.StringPtr("user_home.com#EXT#"),
					UserPrincipalName: to.StringPtr("user_home.com#EXT#@home2.onmicrosoft.com"),
				},
			},
			want: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"user@home.com"},
			},
		},
		{
			name:          "domain case can match",
			kubeGroupName: osaCustomerAdmins,
			want1:         true,
			msGroupMembers: []graphrbac.User{
				{
					Mail: to.StringPtr("user@Home.com"),
				},
			},
			want: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"user@home.com"},
			},
		},
		{
			name:          "user case can't match",
			kubeGroupName: osaCustomerAdmins,
			want1:         true,
			msGroupMembers: []graphrbac.User{
				{
					Mail: to.StringPtr("User@home.com"),
				},
			},
			want: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"User@home.com"},
			},
		},
		//case 5 userPrincipalName matches
		{
			name:          "user with UserPrincipalName field matching (case 5)",
			kubeGroupName: osaCustomerAdmins,
			want1:         true,
			kubeGroup: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{},
			},
			msGroupMembers: []graphrbac.User{
				{
					Mail:              to.StringPtr("notmatching@userprincipalname.com"),
					UserType:          graphrbac.Guest,
					GivenName:         nil,
					UserPrincipalName: to.StringPtr("user@userprincipalname.com"),
				},
			},
			want: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"user@userprincipalname.com"},
			},
		},
		//case 6 - no users matching the Mail or UserPrincipalName yet
		{
			name:          "default, Mail set (case 6)",
			kubeGroupName: osaCustomerAdmins,
			want1:         true,
			kubeGroup: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{},
			},
			msGroupMembers: []graphrbac.User{
				{
					Mail:              to.StringPtr("notmatching@nomatch.com"),
					UserType:          graphrbac.Guest,
					GivenName:         nil,
					UserPrincipalName: to.StringPtr("notmatching2@nomatch.com"),
				},
			},
			want: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"notmatching@nomatch.com"},
			},
		},
		{
			name:          "default, mail not set (case 6)",
			kubeGroupName: osaCustomerAdmins,
			want1:         true,
			kubeGroup: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{},
			},
			msGroupMembers: []graphrbac.User{
				{
					UserType:          graphrbac.Guest,
					GivenName:         nil,
					UserPrincipalName: to.StringPtr("notmatching2@nomatch.com"),
				},
			},
			want: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"notmatching2@nomatch.com"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logrus.NewEntry(logrus.StandardLogger()).WithField("test", tt.name)
			got, got1 := fromMSGraphGroup(log, userv1, tt.kubeGroup, tt.kubeGroupName, tt.msGroupMembers)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fromMSGraphGroup()\n got = %v, \nwant %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("fromMSGraphGroup()\n got1 = %v, \nwant %v", got1, tt.want1)
			}
		})
	}
}
