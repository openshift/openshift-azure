package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/openshift/openshift-azure/pkg/api/admin/api"
)

type Client struct {
	baseURL        string
	subscriptionID string
}

func NewClient(baseURI, subscriptionID string) *Client {
	return &Client{
		baseURL:        baseURI,
		subscriptionID: subscriptionID,
	}
}

func (c *Client) do(method, resourceGroupName, resourceName, subpath string, in, out interface{}) (err error) {
	var inb io.ReadWriter

	if in != nil {
		inb = &bytes.Buffer{}
		if err = json.NewEncoder(inb).Encode(in); err != nil {
			return
		}
	}

	req, err := http.NewRequest(method, c.baseURL+
		"/admin"+
		"/subscriptions/"+c.subscriptionID+
		"/resourceGroups/"+resourceGroupName+
		"/providers/Microsoft.ContainerService/openShiftManagedClusters/"+resourceName+
		subpath, inb)
	if err != nil {
		return
	}

	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%d: %s", resp.StatusCode, string(b))
	}

	if out == nil {
		return
	}

	if out, ok := out.(*[]byte); ok {
		*out = b
		return
	}

	return json.Unmarshal(b, out)
}

func (c *Client) Backup(ctx context.Context, resourceGroupName, resourceName, blobName string) (err error) {
	err = c.do(http.MethodPut, resourceGroupName, resourceName, "/backup/"+blobName, nil, nil)
	return
}

func (c *Client) CreateOrUpdate(ctx context.Context, resourceGroupName, resourceName string, oc *api.OpenShiftManagedCluster) (out *api.OpenShiftManagedCluster, err error) {
	err = c.do(http.MethodPut, resourceGroupName, resourceName, "", oc, &out)
	return
}

func (c *Client) ForceUpdate(ctx context.Context, resourceGroupName, resourceName string) (err error) {
	err = c.do(http.MethodPut, resourceGroupName, resourceName, "/forceUpdate", nil, nil)
	return
}

func (c *Client) Get(ctx context.Context, resourceGroupName, resourceName string) (out *api.OpenShiftManagedCluster, err error) {
	err = c.do(http.MethodGet, resourceGroupName, resourceName, "", nil, &out)
	return
}

func (c *Client) GetControlPlanePods(ctx context.Context, resourceGroupName, resourceName string) (out []byte, err error) {
	err = c.do(http.MethodGet, resourceGroupName, resourceName, "/status", nil, &out)
	return
}

func (c *Client) GetPluginVersion(ctx context.Context, resourceGroupName, resourceName string) (out *api.GenevaActionPluginVersion, err error) {
	err = c.do(http.MethodGet, resourceGroupName, resourceName, "/pluginVersion", nil, &out)
	return
}

func (c *Client) ListClusterVMs(ctx context.Context, resourceGroupName, resourceName string) (out *api.GenevaActionListClusterVMs, err error) {
	err = c.do(http.MethodGet, resourceGroupName, resourceName, "/listClusterVMs", nil, &out)
	return
}

func (c *Client) Reimage(ctx context.Context, resourceGroupName, resourceName, hostname string) (err error) {
	err = c.do(http.MethodPut, resourceGroupName, resourceName, "/reimage/"+hostname, nil, nil)
	return
}

func (c *Client) Restore(ctx context.Context, resourceGroupName, resourceName, blobName string) (err error) {
	err = c.do(http.MethodPut, resourceGroupName, resourceName, "/restore/"+blobName, nil, nil)
	return
}

func (c *Client) RotateSecrets(ctx context.Context, resourceGroupName, resourceName string) (err error) {
	err = c.do(http.MethodPut, resourceGroupName, resourceName, "/rotate/secrets", nil, nil)
	return
}

func (c *Client) RunCommand(ctx context.Context, resourceGroupName, resourceName, hostname, command string) (err error) {
	err = c.do(http.MethodPut, resourceGroupName, resourceName, "/runCommand/"+hostname+"/"+command, nil, nil)
	return
}
