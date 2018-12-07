package openshift

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/openshift/openshift-azure/pkg/fakerp"
	azuretls "github.com/openshift/openshift-azure/pkg/tls"
	"github.com/openshift/openshift-azure/pkg/util/managedcluster"
)

func login(username string) (*api.Config, error) {
	dataDir, err := fakerp.FindDirectory(fakerp.DataDirectory)
	if err != nil {
		return nil, err
	}
	cs, err := managedcluster.ReadConfig(filepath.Join(dataDir, "containerservice.yaml"))
	if err != nil {
		return nil, err
	}

	var password string
	switch username {
	case "customer-cluster-admin":
		password = cs.Config.CustomerAdminPasswd
	case "customer-cluster-reader":
		password = cs.Config.CustomerReaderPasswd
	case "enduser":
		password = cs.Config.EndUserPasswd
	default:
		return nil, fmt.Errorf("unknown username %q", username)
	}

	pool := x509.NewCertPool()
	pool.AddCert(cs.Config.Certificates.Ca.Cert)

	cli := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: pool,
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest(http.MethodGet, "https://"+cs.Properties.FQDN+"/oauth/authorize?response_type=token&client_id=openshift-challenging-client", nil)
	req.Header.Add("X-Csrf-Token", "1")
	req.SetBasicAuth(username, password)

	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}

	location, err := resp.Location()
	if err != nil {
		return nil, err
	}

	fragment, err := url.ParseQuery(location.Fragment)
	if err != nil {
		return nil, err
	}

	return makeKubeConfig(cs.Config.Certificates.Ca.Cert, cs.Properties.FQDN, username, fragment.Get("access_token"))
}

func makeKubeConfig(caCert *x509.Certificate, endpoint, username, token string) (*api.Config, error) {
	clustername := strings.Replace(endpoint, ".", "-", -1)
	authinfoname := username + "/" + clustername
	contextname := "default/" + clustername + "/" + username

	caCertBytes, err := azuretls.CertAsBytes(caCert)
	if err != nil {
		return nil, err
	}

	return &api.Config{
		Clusters: map[string]*api.Cluster{
			clustername: {
				Server:                   "https://" + endpoint,
				CertificateAuthorityData: caCertBytes,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			authinfoname: {
				Token: token,
			},
		},
		Contexts: map[string]*api.Context{
			contextname: {
				Cluster:   clustername,
				Namespace: "default",
				AuthInfo:  authinfoname,
			},
		},
		CurrentContext: contextname,
	}, nil
}
