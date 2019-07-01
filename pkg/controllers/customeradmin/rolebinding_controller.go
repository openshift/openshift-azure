package customeradmin

import (
	"context"
	"reflect"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func addRolebindingController(log *logrus.Entry, m manager.Manager) error {
	options := controller.Options{
		Reconciler: &reconcileRolebinding{
			client: m.GetClient(),
			scheme: m.GetScheme(),
			log:    log,
		},
	}

	c, err := controller.New("customeradmin-rolebinding-controller", m, options)
	if err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &rbacv1.RoleBinding{}}, &handler.EnqueueRequestForObject{})
}

type reconcileRolebinding struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	log    *logrus.Entry
}

var _ reconcile.Reconciler = &reconcileRolebinding{}

// Reconcile receives a rolebinding event and ensures that managed rolebindings
// are not messed with.
// The controller will requeue the request to be processed again if the returned
// error is non-nil or Result.Requeue is true, otherwise upon completion it will
// remove the work from the queue.
func (r *reconcileRolebinding) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	startTime := time.Now()
	metricLabels := prometheus.Labels{"controller": "customeradmin-rolebinding-controller"}

	azureControllersInFlightGauge.With(metricLabels).Inc()
	defer func() {
		azureControllersDurationSummary.With(metricLabels).Observe(time.Now().Sub(startTime).Seconds())
		azureControllersInFlightGauge.With(metricLabels).Dec()
		azureControllersLastExecutedGauge.With(metricLabels).SetToCurrentTime()
	}()

	ctx := context.Background()

	if ignoredNamespace(request.Namespace) {
		return reconcile.Result{}, nil
	}

	if _, found := desiredRolebindings[request.Name]; !found {
		return reconcile.Result{}, nil
	}

	r.log.Debugf("RBC: reconciling rolebinding %s/%s", request.Namespace, request.Name)

	// TODO: can we get away without doing any of this?
	var ns corev1.Namespace
	err := r.client.Get(ctx, types.NamespacedName{Name: request.Namespace}, &ns)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		r.log.Errorf("RBC: error getting namespace: %s", err)
		azureControllersErrorsCounter.With(metricLabels).Inc()
		return reconcile.Result{}, err
	}
	if ns.Status.Phase == corev1.NamespaceTerminating {
		r.log.Debugf("RBC: namespace %s is in phase terminating", request.Namespace)
		return reconcile.Result{}, nil
	}

	var rb rbacv1.RoleBinding
	err = r.client.Get(ctx, request.NamespacedName, &rb)
	switch {
	case err != nil && kerrors.IsNotFound(err):
		r.log.Debugf("RBC: creating rolebinding %s/%s", request.Namespace, request.Name)

		rb = desiredRolebindings[request.Name]
		rb.Namespace = request.Namespace

		err = r.client.Create(ctx, &rb)
		switch {
		case err != nil && kerrors.IsAlreadyExists(err):
			// already exists.  This can happen when the two controllers race,
			// e.g. a namespace relist + the end-user deletes the rolebinding.
		case err != nil:
			r.log.Errorf("RBC: error creating rolebinding: %s", err)
			azureControllersErrorsCounter.With(metricLabels).Inc()
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil

	case err != nil:
		r.log.Errorf("RBC: error getting rolebinding: %s", err)
		azureControllersErrorsCounter.With(metricLabels).Inc()
		return reconcile.Result{}, err
	}

	desired := desiredRolebindings[request.Name]

	// check that our rolebinding matches roleref and subjects
	// TODO: make this more robust
	if rb.RoleRef == desired.RoleRef &&
		reflect.DeepEqual(rb.Subjects, desired.Subjects) &&
		reflect.DeepEqual(rb.Labels, desired.Labels) &&
		reflect.DeepEqual(rb.Annotations, desired.Annotations) {
		return reconcile.Result{}, nil
	}

	rb.RoleRef = desired.RoleRef
	rb.Subjects = desired.Subjects
	rb.Labels = desired.Labels
	rb.Annotations = desired.Annotations

	r.log.Debugf("RBC: updating rolebinding %s/%s", request.Namespace, request.Name)
	err = r.client.Update(ctx, &rb)
	if err != nil {
		r.log.Errorf("RBC: error updating rolebinding: %s", err)
		azureControllersErrorsCounter.With(metricLabels).Inc()
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
