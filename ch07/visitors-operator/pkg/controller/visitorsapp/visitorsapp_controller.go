package visitorsapp

import (
	"context"
	"fmt"
	"time"

	examplev1 "github.com/jdob/visitors-operator/pkg/apis/example/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_visitorsapp")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new VisitorsApp Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileVisitorsApp{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("visitorsapp-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource VisitorsApp
	err = c.Watch(&source.Kind{Type: &examplev1.VisitorsApp{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner VisitorsApp
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &examplev1.VisitorsApp{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &examplev1.VisitorsApp{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileVisitorsApp implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileVisitorsApp{}

// ReconcileVisitorsApp reconciles a VisitorsApp object
type ReconcileVisitorsApp struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a VisitorsApp object and makes changes based on the state read
// and what is in the VisitorsApp.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVisitorsApp) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling VisitorsApp")

	// Fetch the VisitorsApp instance
	v := &examplev1.VisitorsApp{}
	err := r.client.Get(context.TODO(), request.NamespacedName, v)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	var result *reconcile.Result

	// == MySQL ==========
	result, err = r.ensureSecret(request, v, r.mysqlAuthSecret(v))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureDeployment(request, v, r.mysqlDeployment(v))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureService(request, v, r.mysqlService(v))
	if result != nil {
		return *result, err
	}

	mysqlRunning := r.isMysqlUp(v)

	if !mysqlRunning {
		// If MySQL isn't running yet, requeue the reconcile
		// to run again after a delay
		delay := time.Second*time.Duration(5)

		log.Info(fmt.Sprintf("MySQL isn't running, waiting for %s", delay))
		return reconcile.Result{RequeueAfter: delay}, nil
	}

	// == Visitors Backend  ==========
	result, err = r.ensureDeployment(request, v, r.backendDeployment(v))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureService(request, v, r.backendService(v))
	if result != nil {
		return *result, err
	}

	err = r.updateBackendStatus(v)
	if err != nil {
		// Requeue the request if the status could not be updated
		return reconcile.Result{}, err
	}

	result, err = r.handleBackendChanges(v)
	if result != nil {
		return *result, err
	}

	// == Visitors Frontend ==========
	result, err = r.ensureDeployment(request, v, r.frontendDeployment(v))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureService(request, v, r.frontendService(v),
	)
	if result != nil {
		return *result, err
	}

	err = r.updateFrontendStatus(v)
	if err != nil {
		// Requeue the request
		return reconcile.Result{}, err
	}

	result, err = r.handleFrontendChanges(v)
	if result != nil {
		return *result, err
	}

	// == Finish ==========
	// Everything went fine, don't requeue
	return reconcile.Result{}, nil
}
