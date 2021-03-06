// Copyright 2020 IBM Corp.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package remoteresources3

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/gotidy/ptr"
	marketplacev1alpha1 "github.com/redhat-marketplace/redhat-marketplace-operator/pkg/apis/marketplace/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_remoteresources3")

// Add creates a new Node Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRemoteResourceS3{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("remoteresources3-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	labelPreds := []predicate.Predicate{
		predicate.Funcs{
			UpdateFunc: func(evt event.UpdateEvent) bool {
				return true
			},
			CreateFunc: func(evt event.CreateEvent) bool {
				return true
			},
			GenericFunc: func(evt event.GenericEvent) bool {
				return false
			},
			DeleteFunc: func(evt event.DeleteEvent) bool {
				return false
			},
		},
	}

	// Watch for changes to primary resource Node
	err = c.Watch(&source.Kind{Type: &marketplacev1alpha1.RemoteResourceS3{}}, &handler.EnqueueRequestForObject{}, labelPreds...)
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileNode implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRemoteResourceS3{}

// ReconcileNode reconciles a Node object
type ReconcileRemoteResourceS3 struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Node object and makes changes based on the state read
// and what is in the Node.Spec
func (r *ReconcileRemoteResourceS3) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Name", request.Name, "Request.Namespace", request.Namespace)
	reqLogger.Info("Reconciling RemoteResourceS3")

	// Fetch the Node instance
	instance := &marketplacev1alpha1.RemoteResourceS3{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Error(err, "remoteresources3 does not exist")
			return reconcile.Result{}, nil
		}
		reqLogger.Error(err, "Failed to get node")
		return reconcile.Result{}, err
	}

	if instance.Status.Touched == nil {
		instance.Status = marketplacev1alpha1.RemoteResourceS3Status{
			Touched: ptr.Bool(true),
		}
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			return reconcile.Result{}, err
		}
		reqLogger.Info("updated remoteresources3")
	}

	reqLogger.Info("finished reconcile")
	return reconcile.Result{}, nil
}
