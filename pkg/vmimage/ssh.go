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
	session, err := builder.newSSHSession()
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

	_, err = io.Copy(os.Stdout, stdout)
	if err != nil {
		return err
	}

	return nil
}

func (builder *Builder) scp(files []string) error {
	for _, file := range files {
		session, err := builder.newSSHSession()
		if err != nil {
			return err
		}
		defer session.Close()

		stdoutPipe, err := session.StdoutPipe()
		if err != nil {
			return err
		}

		builder.Log.Infof("download %s", file)
		f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		err = session.Start(fmt.Sprintf("sudo cat %s", file))
		if err != nil {
			return err
		}

		io.Copy(f, stdoutPipe)

		err = session.Wait()
		if err != nil {
			return err
		}
		err = session.Close()
		if err == io.EOF {
			continue
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (builder *Builder) newSSHSession() (*ssh.Session, error) {
	signer, err := ssh.NewSignerFromKey(builder.SSHKey)
	if err != nil {
		return nil, err
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

	return client.NewSession()
}
