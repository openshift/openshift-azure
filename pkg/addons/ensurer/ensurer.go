package ensurer

import (
	"context"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	acsapi "github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/client"
	"github.com/openshift/openshift-azure/pkg/log"
)

type SyncPodEnsurer interface {
	EnsureSyncPod(ctx context.Context, m *acsapi.OpenShiftManagedCluster) error
}

type simpleSyncPodEnsurer struct{}

var _ SyncPodEnsurer = &simpleSyncPodEnsurer{}

func NewSimpleSyncPodEnsurer(entry *logrus.Entry) SyncPodEnsurer {
	log.New(entry)
	return &simpleSyncPodEnsurer{}
}

func (spe *simpleSyncPodEnsurer) EnsureSyncPod(ctx context.Context, cs *acsapi.OpenShiftManagedCluster) error {
	kc, err := client.NewKubernetesClientset(ctx, cs.Config.AdminKubeconfig)
	if err != nil {
		return err
	}

	syncNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "openshift-sync",
			Annotations: map[string]string{
				"openshift.io/node-selector": "",
			},
		},
	}

	_, err = kc.CoreV1().Namespaces().Create(syncNamespace)
	switch {
	case err == nil:

	case errors.IsAlreadyExists(err):
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			existingNamespace, err := kc.CoreV1().Namespaces().Get(syncNamespace.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			syncNamespace.ResourceVersion = existingNamespace.ResourceVersion
			_, err = kc.CoreV1().Namespaces().Update(syncNamespace)
			return err
		})
		if err != nil {
			return err
		}

	default:
		return err
	}

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
			Namespace: syncNamespace.Name,
		},
		Data: map[string][]byte{
			"containerservice.yaml": csb,
			"sync.kubeconfig":       kcb,
		},
	}

	_, err = kc.CoreV1().Secrets(syncNamespace.Name).Create(syncSecret)
	switch {
	case err == nil:

	case errors.IsAlreadyExists(err):
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			existingSecret, err := kc.CoreV1().Secrets(syncNamespace.Name).Get(syncSecret.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			syncSecret.ResourceVersion = existingSecret.ResourceVersion
			_, err = kc.CoreV1().Secrets(syncNamespace.Name).Update(syncSecret)
			return err
		})
		if err != nil {
			return err
		}

	default:
		return err
	}

	syncDaemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sync",
			Namespace: "openshift-sync",
		},
		Spec: appsv1.DaemonSetSpec{
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
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": "master-000000",
					},
					HostNetwork: true,
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

	_, err = kc.AppsV1().DaemonSets(syncNamespace.Name).Create(syncDaemonSet)
	switch {
	case err == nil:

	case errors.IsAlreadyExists(err):
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			existingDaemonSet, err := kc.AppsV1().DaemonSets(syncNamespace.Name).Get(syncDaemonSet.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			syncSecret.ResourceVersion = existingDaemonSet.ResourceVersion
			_, err = kc.AppsV1().DaemonSets(syncNamespace.Name).Update(syncDaemonSet)
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
