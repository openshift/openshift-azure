package openshift

import (
	"errors"

	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cli *Client) GetServiceAccountToken(namespace, name string) ([]byte, error) {
	sa, err := cli.CoreV1.ServiceAccounts(namespace).Get(name, meta_v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	for _, ref := range sa.Secrets {
		secret, err := cli.CoreV1.Secrets(namespace).Get(ref.Name, meta_v1.GetOptions{})
		if err != nil {
			return nil, err
		}

		if secret.Type == v1.SecretTypeServiceAccountToken {
			return secret.Data[v1.ServiceAccountTokenKey], nil
		}
	}

	return nil, errors.New("token not found")
}
