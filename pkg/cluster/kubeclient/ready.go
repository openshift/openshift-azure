package kubeclient

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/openshift/openshift-azure/pkg/util/ready"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

func (u *kubeclient) WaitForReadyMaster(ctx context.Context, hostname string) error {
	return wait.PollImmediateUntil(time.Second, func() (bool, error) { return u.masterIsReady(hostname) }, ctx.Done())
}

func (u *kubeclient) masterIsReady(hostname string) (bool, error) {
	r, err := ready.CheckNodeIsReady(u.client.CoreV1().Nodes(), hostname)()
	if !r || err != nil {
		return r, err
	}

	r, err = ready.CheckPodIsReady(u.client.CoreV1().Pods("kube-system"), "master-etcd-"+hostname)()
	if !r || err != nil {
		return r, err
	}

	r, err = ready.CheckPodIsReady(u.client.CoreV1().Pods("kube-system"), "master-api-"+hostname)()
	if !r || err != nil {
		return r, err
	}

	return ready.CheckPodIsReady(u.client.CoreV1().Pods("kube-system"), "controllers-"+hostname)()
}

func (u *kubeclient) WaitForReadyWorker(ctx context.Context, hostname string) error {
	return wait.PollImmediateUntil(time.Second, ready.CheckNodeIsReady(u.client.CoreV1().Nodes(), hostname), ctx.Done())
}

func (u *kubeclient) WaitForReadySyncPod(ctx context.Context) error {
	return wait.PollImmediateUntil(time.Second,
		func() (bool, error) {
			_, err := u.client.CoreV1().
				Services("kube-system").
				ProxyGet("", "sync", "", "/healthz/ready", nil).
				DoRaw()

			switch {
			case err == nil:
				return true, nil
			case errors.IsServiceUnavailable(err):
				u.log.Info("pod not yet started")
				return false, nil
			case errors.IsInternalError(err):
				if err, ok := err.(*errors.StatusError); ok && err.ErrStatus.Details != nil && len(err.ErrStatus.Details.Causes) == 1 {
					u.log.Info(err.ErrStatus.Details.Causes[0].Message)
				}
				return false, nil
			default:
				return false, err
			}
		},
		ctx.Done())
}
