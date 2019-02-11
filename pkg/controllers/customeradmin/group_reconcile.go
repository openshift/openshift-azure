package customeradmin

import (
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	userv1client "github.com/openshift/client-go/user/clientset/versioned/typed/user/v1"
	"github.com/sirupsen/logrus"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func updateKubeGroup(log *logrus.Entry, userV1 userv1client.UserV1Interface, kubeGroupName string, msGroupMembers []graphrbac.User) error {
	kubeGroup, err := userV1.Groups().Get(kubeGroupName, meta_v1.GetOptions{})
	if err != nil && !kerrors.IsNotFound(err) {
		return err
	}
	if err != nil && kerrors.IsNotFound(err) {
		// for some reason when IsNotFound kubeGroup is not nil and we go through
		// update path which won't work when the group does not exist.
		kubeGroup = nil
	}
	kubeGroupDef, changed := fromMSGraphGroup(log, kubeGroup, kubeGroupName, []graphrbac.User{})
	if kubeGroup == nil {
		log.Debugf("Creating new kube group %s", kubeGroupName)
		_, err = userV1.Groups().Create(kubeGroupDef)
		if err != nil {
			return err
		}
	} else if changed {
		log.Debugf("Updating existing kube group %s", kubeGroupName)
		_, err = userV1.Groups().Update(kubeGroupDef)
		if err != nil {
			return err
		}
	}
	return nil
}

func reconcileGroups(log *logrus.Entry, gc *graphrbac.GroupsClient, userV1 userv1client.UserV1Interface, groupMap map[string]string) error {
	aadGroupID, have := groupMap[osaCustomerAdmins]
	if !have {
		// CustomerAdminGroupID not configured: ensure the group is empty
		err := updateKubeGroup(log, userV1, osaCustomerAdmins, []graphrbac.User{})
		if err != nil {
			return err
		}
	} else {
		msGroupMembers, err := getAADGroupMembers(gc, aadGroupID)
		if err != nil {
			return err
		}
		err = updateKubeGroup(log, userV1, osaCustomerAdmins, msGroupMembers)
		if err != nil {
			return err
		}
	}

	aadGroupID, have = groupMap[osaCustomerReaders]
	if !have {
		// CustomerReaderGroupID not configured: ensure the group is empty
		err := updateKubeGroup(log, userV1, osaCustomerReaders, []graphrbac.User{})
		if err != nil {
			return err
		}
	} else {
		msGroupMembers, err := getAADGroupMembers(gc, aadGroupID)
		if err != nil {
			return err
		}
		err = updateKubeGroup(log, userV1, osaCustomerReaders, msGroupMembers)
		if err != nil {
			return err
		}
	}

	return nil
}
