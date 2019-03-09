package vmimage

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

func (builder *Builder) ssh() error {
	signer, err := ssh.NewSignerFromKey(builder.SSHKey)
	if err != nil {
		return err
	}

	config := &ssh.ClientConfig{
		User: adminUsername,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil },
		Timeout:         10 * time.Second,
	}

	var client *ssh.Client
	t := time.NewTicker(10 * time.Second)
	for {
		client, err = ssh.Dial("tcp", fmt.Sprintf("%s.%s.cloudapp.azure.com:22", builder.DomainNameLabel, builder.Location), config)
		if err == nil {
			break
		}
		<-t.C
	}
	t.Stop()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}

	err = session.Start("sudo tail -F -n +1 /var/lib/waagent/custom-script/download/0/stdout")
	if err != nil {
		return err
	}

	io.Copy(os.Stdout, stdout)

	return nil
}
