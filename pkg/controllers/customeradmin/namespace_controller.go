package customeradmin

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func addNamespaceController(log *logrus.Entry, m manager.Manager) error {
	options := controller.Options{
		Reconciler: &reconcileNamespace{
			client: m.GetClient(),
			scheme: m.GetScheme(),
			log:    log,
		},
	}

	c, err := controller.New("customeradmin-namespace-controller", m, options)
	if err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &corev1.Namespace{}}, &handler.EnqueueRequestForObject{})
}

type reconcileNamespace struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	log    *logrus.Entry
}

var _ reconcile.Reconciler = &reconcileNamespace{}

// Reconcile receives a namespace event and ensures that all the necessary
// customer-admin rolebindings are created.
// The controller will requeue the request to be processed again if the returned
// error is non-nil or Result.Requeue is true, otherwise upon completion it will
// remove the work from the queue.
func (r *reconcileNamespace) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	startTime := time.Now()
	metricLabels := prometheus.Labels{"controller": "customeradmin-namespace-controller"}

	azureControllersInFlightGauge.With(metricLabels).Inc()
	defer func() {
		azureControllersDurationSummary.With(metricLabels).Observe(time.Now().Sub(startTime).Seconds())
		azureControllersInFlightGauge.With(metricLabels).Dec()
		azureControllersLastExecutedGauge.With(metricLabels).SetToCurrentTime()
	}()

	ctx := context.Background()

	if ignoredNamespace(request.Name) {
		return reconcile.Result{}, nil
	}

	r.log.Debugf("NSC: reconciling namespace %s", request.Name)

	var ns corev1.Namespace
	err := r.client.Get(ctx, request.NamespacedName, &ns)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		r.log.Errorf("NSC: error getting namespace: %s", err)
		azureControllersErrorsCounter.With(metricLabels).Inc()
		return reconcile.Result{}, err
	}
	if ns.Status.Phase == corev1.NamespaceTerminating {
		r.log.Debugf("NSC: namespace %s is in phase terminating", request.Name)
		return reconcile.Result{}, nil
	}

	for _, desired := range desiredRolebindings {
		desired.Namespace = request.Name

		r.log.Debugf("NSC: creating rolebinding %s/%s", desired.Namespace, desired.Name)
		err = r.client.Create(ctx, &desired)
		switch {
		case err != nil && kerrors.IsAlreadyExists(err):
			// already exists.  This can happen when the two controllers race,
			// e.g. a namespace relist + the end-user deletes the rolebinding.
		case err != nil:
			r.log.Infof("NSC: error creating rolebinding: %s", err)
			azureControllersErrorsCounter.With(metricLabels).Inc()
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}
