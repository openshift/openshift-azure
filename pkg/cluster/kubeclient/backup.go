package kubeclient

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/openshift-azure/pkg/util/ready"
	"github.com/openshift/openshift-azure/pkg/util/wait"
)

func (u *kubeclient) BackupCluster(ctx context.Context, backupName string) error {
	u.log.Infof("running an etcd backup")
	cronjob, err := u.client.BatchV1beta1().CronJobs("openshift-etcd").Get("etcd-backup", metav1.GetOptions{})
	if err != nil {
		return err
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupName,
			Namespace: cronjob.Namespace,
		},
		Spec: cronjob.Spec.JobTemplate.Spec,
	}

	job.Spec.BackoffLimit = to.Int32Ptr(0)
	job.Spec.Template.Spec.Containers[0].Args = []string{fmt.Sprintf("-blobname=%s", backupName), "save"}

	job, err = u.client.BatchV1().Jobs(job.Namespace).Create(job)
	if err != nil {
		return err
	}

	defer func() {
		err = u.client.BatchV1().Jobs(job.Namespace).Delete(job.Name, &metav1.DeleteOptions{})
		if err != nil {
			u.log.Infof("failed to delete job: %s", job.Name)
		}
	}()

	err = wait.PollImmediateUntil(2*time.Second, func() (bool, error) {
		return ready.CheckJobIsReady(u.client.BatchV1().Jobs(job.Namespace), job.Name)()
	}, ctx.Done())
	if err != nil {
		return err
	}

	// TODO: verify that the backup blob was created/exists and has bytes
	if job.Status.Failed > 0 {
		return fmt.Errorf("backup pod failed: %s", job.Name)
	}

	return nil
}
