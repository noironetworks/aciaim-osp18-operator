/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	apiv1alpha1 "github.com/noironetworks/aciaim-osp18-operator/api/v1alpha1"
)

// CiscoAciAimReconciler reconciles a CiscoAciAim object
type CiscoAciAimReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// GetLog returns a logger object with a prefix of "conroller.name" and aditional controller context fields
func (r *CiscoAciAimReconciler) GetLogger(ctx context.Context) logr.Logger {
	return log.FromContext(ctx).WithName("Controllers").WithName("CiscoAciAimController")
}


// +kubebuilder:rbac:groups=api.cisco.com,resources=ciscoaciaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=api.cisco.com,resources=ciscoaciaims/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=api.cisco.com,resources=ciscoaciaims/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CiscoAciAim object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *CiscoAciAimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)
    //Fetch the ciscoAciAim instance that has to be reconciled
	instance := &ciscoaciaimv1.CiscoAciAim{}

	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected.
			// For additional cleanup logic use finalizers. Return and don't requeue.
			Log.Info("CiscoAciAim instance not found, probably deleted before reconciled. Nothing to do.")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		Log.Error(err, "Failed to read the Cisco Aci Aim instance.")
		return ctrl.Result{}, err
	}
    replicas := int(*instance.Spec.Replicas)

    // Fetch the Deployment object that needs scaling
    deployment := &appsv1.Deployment{}
    err = r.Client.Get(ctx, types.NamespacedName{Name: instance.Spec.DeploymentName, Namespace: instance.Namespace}, deployment)
    if err != nil {
        if k8s_errors.IsNotFound(err) {
            log.Info("Deployment not found, could be deleted.")
            return ctrl.Result{}, nil
        }
        log.Error(err, "Failed to fetch Deployment.")
        return ctrl.Result{}, err
    }

    // Check if the current number of replicas matches the desired number
    if *deployment.Spec.Replicas != replicas {
        log.Info("Updating Deployment replicas", "Current", *deployment.Spec.Replicas, "Desired", replicas)
        deployment.Spec.Replicas = &replicas
        err = r.Client.Update(ctx, deployment)
        if err != nil {
            log.Error(err, "Failed to update Deployment replicas.")
            return ctrl.Result{}, err
        }
    }    

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CiscoAciAimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.CiscoAciAim{}).
		Complete(r)
}
