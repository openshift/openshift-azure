package openshift

import (
	"time"

	templatev1 "github.com/openshift/api/template/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/pkg/util/ready"
)

func (cli *Client) InstantiateTemplate(srcTemplateName, dstNamespace string) error {
	template, err := cli.TemplateV1.Templates("openshift").Get(srcTemplateName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	_, err = cli.TemplateV1.TemplateInstances(dstNamespace).Create(
		&templatev1.TemplateInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name: dstNamespace,
			},
			Spec: templatev1.TemplateInstanceSpec{
				Template: *template,
			},
		})
	if err != nil {
		return err
	}

	return wait.PollImmediate(2*time.Second, 20*time.Minute, ready.CheckTemplateInstanceIsReady(cli.TemplateV1.TemplateInstances(dstNamespace), dstNamespace))
}
