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
	userv1 := fakeuserv1.NewSimpleClientset(
		&v1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "live.com#foo@bar.com",
			},
		},
		&v1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "bar@foo.com",
			},
		},
		&v1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "live.com#user@homecorp.com",
			},
		},
		&v1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "user@nativecorp.com",
			},
		},
		&v1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "user@guestcorp.com",
			},
		},
		&v1.User{
			ObjectMeta: metav1.ObjectMeta{
				Name: "user@amazingcorp.com",
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
			name:          "guest member reconcile external tree",
			kubeGroupName: osaCustomerAdmins,
			want1:         true,
			kubeGroup: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"live.com#user@homecorp.com", "user@nativecorp.com", "user@guestcorp.com"},
			},
			msGroupMembers: []graphrbac.User{
				{
					// ocpUser: live.com#user@homecorp.com
					Mail:              to.StringPtr("user@homecorp.com"),
					UserType:          graphrbac.Guest,
					MailNickname:      to.StringPtr("user_homecorp.com#EXT#"),
					UserPrincipalName: to.StringPtr("user_homecorp.com#EXT#@nativecorp.onmicrosoft.com"),
				},
				{
					// ocpUser: user@guestcorp.com
					Mail:              to.StringPtr("user@guestcorp.com"),
					UserType:          graphrbac.Guest,
					MailNickname:      to.StringPtr("user_guestcorp.com#EXT#"),
					UserPrincipalName: to.StringPtr("user_guestcorp.com#EXT#@nativecorp.onmicrosoft.com"),
				},
				{
					// ocpUser: user@nativecorp.com
					Mail:              nil,
					UserType:          graphrbac.Member,
					MailNickname:      to.StringPtr("user"),
					UserPrincipalName: to.StringPtr("user@nativecorp.com"),
				},
				// TOFIX: This is usecase which I seen, but without customer I cant replicate without
				// customer help at the moment.
				//{
				//	// ocpUser: user@amazingcorp.com
				//	Mail:              nil,
				//	UserType:          graphrbac.Guest,
				//	MailNickname:      to.StringPtr("user@amazingcorp.com"),
				//	UserPrincipalName: to.StringPtr("user_amazingcorp.com#EXT#@foo.onmicrosoft.com"),
				//},
			},
			want: &v1.Group{
				ObjectMeta: meta_v1.ObjectMeta{
					Name: osaCustomerAdmins,
				},
				Users: []string{"live.com#user@homecorp.com", "user@guestcorp.com", "user@nativecorp.com"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logrus.NewEntry(logrus.StandardLogger()).WithField("test", tt.name)
			got, got1 := fromMSGraphGroup(log, userv1, tt.kubeGroup, tt.kubeGroupName, tt.msGroupMembers)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fromMSGraphGroup() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("fromMSGraphGroup() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
