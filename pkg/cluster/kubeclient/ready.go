package kubeclient

import (
	"context"
	"syscall"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	utilerrors "github.com/openshift/openshift-azure/pkg/util/errors"
	"github.com/openshift/openshift-azure/pkg/util/ready"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

func (u *Kubeclientset) WaitForReadyMaster(ctx context.Context, hostname string) error {
	return wait.PollImmediateUntil(time.Second, func() (bool, error) { return u.masterIsReady(hostname) }, ctx.Done())
}

func (u *Kubeclientset) masterIsReady(hostname string) (bool, error) {
	r, err := ready.CheckNodeIsReady(u.Client.CoreV1().Nodes(), hostname)()
	if !r || err != nil {
		return r, err
	}

	r, err = ready.CheckPodIsReady(u.Client.CoreV1().Pods("kube-system"), "master-etcd-"+hostname)()
	if !r || err != nil {
		return r, err
	}

	r, err = ready.CheckPodIsReady(u.Client.CoreV1().Pods("kube-system"), "master-api-"+hostname)()
	if !r || err != nil {
		return r, err
	}

	return ready.CheckPodIsReady(u.Client.CoreV1().Pods("kube-system"), "controllers-"+hostname)()
}

func (u *Kubeclientset) WaitForReadyWorker(ctx context.Context, hostname string) error {
	return wait.PollImmediateUntil(time.Second, ready.CheckNodeIsReady(u.Client.CoreV1().Nodes(), hostname), ctx.Done())
}

func (u *Kubeclientset) WaitForReadySyncPod(ctx context.Context) error {
	return wait.PollImmediateUntil(10*time.Second,
		func() (bool, error) {
			d, err := u.Client.AppsV1().Deployments("kube-system").Get("sync", metav1.GetOptions{})
			switch {
			case errors.IsNotFound(err):
				return false, nil
			case err != nil:
				return false, err
			}

			isReady := ready.DeploymentIsReady(d)
			if isReady {
				return true, nil
			}

			_, err = u.Client.CoreV1().
				Services("kube-system").
				ProxyGet("", "sync", "", "/healthz/ready", nil).
				DoRaw()

			switch {
			case errors.IsServiceUnavailable(err):
				u.Log.Info("pod not yet started")
				err = nil
			case errors.IsInternalError(err):
				if err, ok := err.(*errors.StatusError); ok && err.ErrStatus.Details != nil && len(err.ErrStatus.Details.Causes) == 1 {
					u.Log.Info(err.ErrStatus.Details.Causes[0].Message)
				}
				err = nil
			case utilerrors.IsMatchingSyscallError(err, syscall.ECONNREFUSED):
				u.Log.Infof("WaitForReadySyncPod: will retry on the following error %v", err)
				err = nil
			}

			return false, err
		},
		ctx.Done())
}
