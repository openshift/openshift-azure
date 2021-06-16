package kubeclient

import (
	"context"
	"encoding/hex"

	"github.com/Azure/go-autorest/autorest/to"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
)

func (u *Kubeclientset) EnsureSyncPod(ctx context.Context, syncImage string, hash []byte) error {
	{
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sync",
				Namespace: "kube-system",
			},
		}
		_, err := u.Client.CoreV1().ServiceAccounts(sa.Namespace).Create(sa)
		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	}

	{
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			existing, err := u.Seccli.SecurityV1().SecurityContextConstraints().Get("privileged", metav1.GetOptions{})
			if err != nil {
				return err
			}
			for _, user := range existing.Users {
				if user == "system:serviceaccount:kube-system:sync" {
					return nil
				}
			}
			existing.Users = append(existing.Users, "system:serviceaccount:kube-system:sync")
			_, err = u.Seccli.SecurityV1().SecurityContextConstraints().Update(existing)
			return err
		})
		if err != nil {
			return err
		}
	}

	{
		d := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sync",
				Namespace: "kube-system",
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
						Annotations: map[string]string{
							"checksum": hex.EncodeToString(hash),
						},
						Labels: map[string]string{
							"app": "sync",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Args:            []string{"sync"},
								Image:           syncImage,
								ImagePullPolicy: corev1.PullAlways,
								Name:            "sync",
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 8080,
									},
								},
								ReadinessProbe: &corev1.Probe{
									Handler: corev1.Handler{
										HTTPGet: &corev1.HTTPGetAction{
											Path: "/healthz/ready",
											Port: intstr.FromInt(8080),
										},
									},
									InitialDelaySeconds: 30,
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										MountPath: "/_data/_out",
										Name:      "master-cloud-provider",
										ReadOnly:  true,
									},
								},
								Env: []corev1.EnvVar{
									{
										Name: "masterNodeName",
										ValueFrom: &corev1.EnvVarSource{
											FieldRef: &corev1.ObjectFieldSelector{
												FieldPath: "spec.nodeName"},
										},
									},
								},
							},
						},
						NodeSelector: map[string]string{
							"node-role.kubernetes.io/master": "true",
						},
						ServiceAccountName: "sync",
						Volumes: []corev1.Volume{
							{
								Name: "master-cloud-provider",
								VolumeSource: corev1.VolumeSource{
									HostPath: &corev1.HostPathVolumeSource{
										Path: "/etc/origin/cloudprovider",
									},
								},
							},
						},
					},
				},
				Strategy: appsv1.DeploymentStrategy{
					Type: appsv1.RecreateDeploymentStrategyType,
				},
			},
		}

		_, err := u.Client.AppsV1().Deployments(d.Namespace).Create(d)
		if errors.IsAlreadyExists(err) {
			err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				existing, err := u.Client.AppsV1().Deployments(d.Namespace).Get(d.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				d.ResourceVersion = existing.ResourceVersion
				_, err = u.Client.AppsV1().Deployments(d.Namespace).Update(d)
				return err
			})
		}
		if err != nil {
			return err
		}
	}

	{
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sync",
				Namespace: "kube-system",
				Labels: map[string]string{
					"app": "sync",
				},
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Name: "http",
						Port: 8080,
					},
				},
				Selector: map[string]string{
					"app": "sync",
				},
				PublishNotReadyAddresses: true,
			},
		}

		_, err := u.Client.CoreV1().Services(svc.Namespace).Create(svc)
		if errors.IsAlreadyExists(err) {
			err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				existing, err := u.Client.CoreV1().Services(svc.Namespace).Get(svc.Name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				svc.ResourceVersion = existing.ResourceVersion
				svc.Spec.ClusterIP = existing.Spec.ClusterIP
				_, err = u.Client.CoreV1().Services(svc.Namespace).Update(svc)
				return err
			})
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func (u *Kubeclientset) RemoveSyncPod(ctx context.Context) error {
	return u.Client.AppsV1().Deployments("kube-system").Delete("sync", &metav1.DeleteOptions{})
}

func (u *Kubeclientset) RemoveValidatingWebhookConfiguration(ctx context.Context) error {
	return u.Client.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Delete("aro-admission-controller.aro.openshift.io", &metav1.DeleteOptions{})
}
