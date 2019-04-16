package customeradmin

import (
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/to"
	v1 "github.com/openshift/api/user/v1"
	"github.com/sirupsen/logrus"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFromMSGraphGroup(t *testing.T) {
	tests := []struct {
		name           string
		kubeGroup      *v1.Group
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logrus.NewEntry(logrus.StandardLogger()).WithField("test", tt.name)
			got, got1 := fromMSGraphGroup(log, tt.kubeGroup, tt.kubeGroupName, tt.msGroupMembers)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fromMSGraphGroup() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("fromMSGraphGroup() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
