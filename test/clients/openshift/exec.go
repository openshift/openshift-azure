package openshift

import (
	"net/url"

	"k8s.io/client-go/tools/remotecommand"
)

func (cli *Client) CommandExecutor(requestMethod string, url *url.URL) (remotecommand.Executor, error) {
	return remotecommand.NewSPDYExecutor(cli.config, requestMethod, url)
}
