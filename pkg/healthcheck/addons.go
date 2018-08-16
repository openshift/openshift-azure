package healthcheck

import (
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
)

// TODO: consider if this should be a separate function in the plugin interface
func ensureSyncPod(kc *kubernetes.Clientset, cs *acsapi.OpenShiftManagedCluster) error {
	csb, err := yaml.Marshal(cs)
	if err != nil {
		return err
	}

	kcb, err := yaml.Marshal(cs.Config.SyncKubeconfig)
	if err != nil {
		return err
	}

	// TODO: Hash content and add as an annotation in the deployment template
	syncSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sync",
			Namespace: "openshift-infra",
		},
		Data: map[string][]byte{
			"containerservice.yaml": csb,
			"sync.kubeconfig":       kcb,
		},
	}

	_, err = kc.CoreV1().Secrets(syncSecret.Namespace).Create(syncSecret)
	switch {
	case err == nil:

	case errors.IsAlreadyExists(err):
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			existingSecret, err := kc.CoreV1().Secrets(syncSecret.Namespace).Get(syncSecret.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			syncSecret.ResourceVersion = existingSecret.ResourceVersion
			_, err = kc.CoreV1().Secrets(syncSecret.Namespace).Update(syncSecret)
			return err
		})
		if err != nil {
			return err
		}

	default:
		return err
	}

	syncDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sync",
			Namespace: "openshift-infra",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: to.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "sync",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "sync",
					},
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: to.BoolPtr(false),
					Containers: []corev1.Container{
						{
							Name:            "sync",
							Image:           cs.Config.SyncImage,
							ImagePullPolicy: corev1.PullAlways,
							Args:            []string{"--config=/_data/containerservice.yaml"},
							Env: []corev1.EnvVar{
								{
									Name:  "KUBECONFIG",
									Value: "/_data/sync.kubeconfig",
								},
								/*
									{
										Name:  "AZURE_TENANT_ID",
										Value: cs.Config.TenantID,
									},
									{
										Name:  "AZURE_SUBSCRIPTION_ID",
										Value: cs.Config.SubscriptionID,
									},
									{
										Name:  "RESOURCEGROUP",
										Value: cs.Config.ResourceGroup,
									},
								*/
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/_data",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "sync",
								},
							},
						},
					},
				},
			},
		},
	}

	_, err = kc.AppsV1().Deployments(syncDeployment.Namespace).Create(syncDeployment)
	switch {
	case err == nil:

	case errors.IsAlreadyExists(err):
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			existingDeployment, err := kc.AppsV1().Deployments(syncDeployment.Namespace).Get(syncDeployment.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			syncSecret.ResourceVersion = existingDeployment.ResourceVersion
			_, err = kc.AppsV1().Deployments(syncDeployment.Namespace).Update(syncDeployment)
			return err
		})
		if err != nil {
			return err
		}

	default:
		return err
	}

	return nil
}
