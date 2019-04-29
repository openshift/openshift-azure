package customeradmin

import (
	"context"
	"io/ioutil"
	"time"

	"github.com/ghodss/yaml"
	userv1client "github.com/openshift/client-go/user/clientset/versioned/typed/user/v1"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/graphrbac"
)

const (
	osaCustomerAdmins = "osa-customer-admins"
)

type reconcileGroup struct {
	userV1    userv1client.UserV1Interface
	aadClient graphrbac.RBACGroupsClient
	log       *logrus.Entry
	groupMap  map[string]string
	config    api.AADIdentityProvider
}

var _ reconcile.Reconciler = &reconcileGroup{}

func addGroupController(ctx context.Context, log *logrus.Entry, m manager.Manager, stopCh <-chan struct{}) error {
	r := &reconcileGroup{
		log:      log,
		groupMap: map[string]string{},
	}
	err := r.load("_data/_out/aad-group-sync.yaml")
	if err != nil {
		return err
	}

	r.userV1 = userv1client.NewForConfigOrDie(m.GetConfig())

	r.aadClient, err = newAADGroupsClient(ctx, log, r.config)
	if err != nil {
		return err
	}

	c, err := controller.New("customeradmin-group-controller", m, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	events := make(chan event.GenericEvent)
	timerSource := source.Channel{Source: events}
	ticker := time.NewTicker(60 * time.Second)
	timerSource.InjectStopChannel(stopCh)
	go func() {
		for {
			select {
			case <-ticker.C:
				events <- event.GenericEvent{}
			case <-stopCh:
				log.Info("shutting down ticker")
				ticker.Stop()
				return
			}
		}
	}()
	return c.Watch(&timerSource, &handler.EnqueueRequestForObject{}, &predicate.Funcs{GenericFunc: r.pollEvent})
}

func (r *reconcileGroup) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// not actually used (pollEvent is the real callback) but the controller.New() really wants it.
	return reconcile.Result{}, nil
}

func (r *reconcileGroup) load(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(b, &r.config); err != nil {
		return err
	}
	if r.config.CustomerAdminGroupID != nil {
		r.groupMap[osaCustomerAdmins] = *r.config.CustomerAdminGroupID
	}
	return nil
}

func (r *reconcileGroup) pollEvent(event.GenericEvent) bool {
	r.log.Debug("AAD Group Reconciler (poll)..")
	err := reconcileGroups(r.log, r.aadClient, r.userV1, r.groupMap)
	if err != nil {
		r.log.Error(err)
	}

	return err == nil
}
