package fakerp

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/cluster/names"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

type ssher struct {
	pipcli    azureclient.PublicIPAddressesClient
	cs        *api.OpenShiftManagedCluster
	masterIPs []string
}

func newSSHer(ctx context.Context, log *logrus.Entry, cs *api.OpenShiftManagedCluster) (*ssher, error) {
	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
	if err != nil {
		return nil, err
	}

	s := &ssher{
		pipcli: azureclient.NewPublicIPAddressesClient(ctx, log, cs.Properties.AzProfile.SubscriptionID, authorizer),
		cs:     cs,
	}

	ips, err := s.pipcli.ListVirtualMachineScaleSetPublicIPAddressesComplete(ctx, s.cs.Properties.AzProfile.ResourceGroup, "ss-master")
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
	signer, err := ssh.NewSignerFromKey(s.cs.Config.SSHKey)
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
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}

	err = session.Start(cmd)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, stdout)
	return err
}
