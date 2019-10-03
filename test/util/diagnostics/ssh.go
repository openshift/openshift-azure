package diagnostics

import (
	"context"
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/names"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/network"
)

type ssher struct {
	pipcli    network.PublicIPAddressesClient
	masterIPs []string
	sshkey    *rsa.PrivateKey
}

func NewSSHer(ctx context.Context, log *logrus.Entry, subscriptionID, resourceGroup string, sshkey *rsa.PrivateKey) (*ssher, error) {
	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
	if err != nil {
		return nil, err
	}

	s := &ssher{
		pipcli: network.NewPublicIPAddressesClient(ctx, log, subscriptionID, authorizer),
		sshkey: sshkey,
	}

	ips, err := s.pipcli.ListVirtualMachineScaleSetPublicIPAddressesComplete(ctx, resourceGroup, "ss-master")
	if err != nil {
		return nil, err
	}

	for ips.NotDone() {
		s.masterIPs = append(s.masterIPs, *ips.Value().IPAddress)

		err = ips.Next()
		if err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *ssher) dialViaMaster(ctx context.Context, config *ssh.ClientConfig, addr string) (net.Conn, error) {
	for _, ip := range s.masterIPs {
		client, err := ssh.Dial("tcp", ip+":22", config)
		if err != nil {
			continue
		}
		return client.Dial("tcp", addr)
	}

	return nil, fmt.Errorf("couldn't dial any master")
}

func (s *ssher) Dial(ctx context.Context, hostname string) (*ssh.Client, error) {
	signer, err := ssh.NewSignerFromKey(s.sshkey)
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User: "cloud-user",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil },
		Timeout:         10 * time.Second,
	}

	switch names.GetAgentRole(hostname) {
	case api.AgentPoolProfileRoleMaster:
		_, instanceID, err := names.GetScaleSetNameAndInstanceID(hostname)
		if err != nil {
			return nil, err
		}

		instance, err := strconv.ParseUint(instanceID, 36, 64)
		if err != nil {
			return nil, err
		}

		if int(instance) > len(s.masterIPs) {
			return nil, fmt.Errorf("couldn't find IP for master")
		}

		return ssh.Dial("tcp", s.masterIPs[int(instance)]+":22", config)

	default:
		conn, err := s.dialViaMaster(ctx, config, hostname+":22")
		if err != nil {
			return nil, err
		}

		c, chans, reqs, err := ssh.NewClientConn(conn, hostname, config)
		if err != nil {
			return nil, err
		}

		return ssh.NewClient(c, chans, reqs), nil
	}
}

func (s *ssher) RunRemoteCommandAndSaveToFile(client *ssh.Client, cmd, filename string) error {
	result, err := s.RunRemoteCommand(client, cmd)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, result, 0666)
}

func (s *ssher) RunRemoteCommand(client *ssh.Client, cmd string) ([]byte, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = session.Start(cmd)
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(stdout)
	return b, err
}
