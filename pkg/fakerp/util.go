package fakerp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	v20190831 "github.com/openshift/openshift-azure/pkg/api/2019-08-31"
	admin "github.com/openshift/openshift-azure/pkg/api/admin"
	"github.com/openshift/openshift-azure/pkg/util/derived"
	"github.com/openshift/openshift-azure/pkg/util/tls"
)

func (s *Server) badRequest(w http.ResponseWriter, msg string) {
	resp := fmt.Sprintf("400 Bad Request: %s", msg)
	s.log.Debug(resp)
	http.Error(w, resp, http.StatusBadRequest)
}

func isAdminRequest(req *http.Request) bool {
	// TODO: Align with the production RP once it supports the admin API
	return strings.HasPrefix(req.URL.Path, "/admin")
}

// adminreply returns admin requests data
func (s *Server) adminreply(w http.ResponseWriter, err error, out interface{}) {
	if err != nil {
		s.badRequest(w, err.Error())
		return
	}

	if out == nil {
		return
	}

	if b, ok := out.([]byte); ok {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(b)
		return
	}

	b, err := json.Marshal(out)
	if err != nil {
		s.badRequest(w, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
	return
}

// reply return either admin or external api response
func (s *Server) reply(w http.ResponseWriter, req *http.Request, cs *api.OpenShiftManagedCluster) {
	var res []byte
	var err error
	if isAdminRequest(req) {
		oc := admin.FromInternal(cs)
		res, err = json.Marshal(oc)
	} else {
		oc := v20190831.FromInternal(cs)
		res, err = json.Marshal(&oc)
	}
	if err != nil {
		s.badRequest(w, fmt.Sprintf("Failed to marshal response: %v", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(res)
}

func writeHelpers(log *logrus.Entry, cs *api.OpenShiftManagedCluster) error {
	b, err := derived.MasterCloudProviderConf(cs, true)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_data/_out/azure.conf", b, 0600)
	if err != nil {
		return err
	}

	b, err = derived.AadGroupSyncConf(cs)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("_data/_out/aad-group-sync.yaml", b, 0600)
	if err != nil {
		return err
	}

	b, err = tls.PrivateKeyAsBytes(cs.Config.SSHKey)
	if err != nil {
		return err
	}
	// ensure both the new key and the old key are on disk so
	// you can SSH in regardless of the state of a VM after an update
	if _, err = os.Stat("_data/_out/id_rsa"); err == nil {
		oldb, err := ioutil.ReadFile("_data/_out/id_rsa")
		if err != nil {
			return err
		}
		if !bytes.Equal(b, oldb) {
			err = ioutil.WriteFile("_data/_out/id_rsa.old", oldb, 0600)
			if err != nil {
				return err
			}
		}
	}
	err = ioutil.WriteFile("_data/_out/id_rsa", b, 0600)
	if err != nil {
		return err
	}

	b, err = yaml.Marshal(cs.Config.AdminKubeconfig)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("_data/_out/admin.kubeconfig", b, 0600)
}
