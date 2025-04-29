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
    ciscoaciaimv1 "github.com/noironetworks/aciaim-osp18-operator/api/v1alpha1"
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
// +kubebuilder:rbac:groups=api.cisco.com,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mariadb.openstack.org,resources=galeras,verbs=get;list;watch
// +kubebuilder:rbac:groups=rabbitmq.openstack.org,resources=rabbitmqclusters,verbs=get;list;watch


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

    // --- MariaDB Readiness ---
    var mariadb mariadbv1.Galera
    if err := r.Get(ctx, types.NamespacedName{Name: aim.Spec.MariaDBInstance, Namespace: aim.Namespace}, &mariadb); err != nil {
        setCondition(&aim, "MariaDBReady", metav1.ConditionFalse, "NotFound", "MariaDB instance not found")
        return r.updateStatusAndRequeue(ctx, &aim)
    }
    if !mariadb.Status.Ready {
        setCondition(&aim, "MariaDBReady", metav1.ConditionFalse, "NotReady", "MariaDB not ready")
        return r.updateStatusAndRequeue(ctx, &aim)
    }

    // --- RabbitMQ Readiness ---
    var rabbitmq rabbitmqv1.RabbitmqCluster
    if err := r.Get(ctx, types.NamespacedName{Name: aim.Spec.RabbitMQCluster, Namespace: aim.Namespace}, &rabbitmq); err != nil {
        setCondition(&aim, "RabbitMQReady", metav1.ConditionFalse, "NotFound", "RabbitMQ cluster not found")
        return r.updateStatusAndRequeue(ctx, &aim)
    }
    if !rabbitmq.Status.Ready {
        setCondition(&aim, "RabbitMQReady", metav1.ConditionFalse, "NotReady", "RabbitMQ not ready")
        return r.updateStatusAndRequeue(ctx, &aim)
    }

    // --- Dependencies are ready, deploy AIM ---
    setCondition(&aim, "MariaDBReady", metav1.ConditionTrue, "Ready", "MariaDB is ready")
    setCondition(&aim, "RabbitMQReady", metav1.ConditionTrue, "Ready", "RabbitMQ is ready")
    aim.Status.Ready = true
    if err := r.Status().Update(ctx, &aim); err != nil {
        return ctrl.Result{}, err
    }


    replicas := int(*instance.Spec.Replicas)


    // Define Deployment
    deployment := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      deployer.Name,
            Namespace: deployer.Namespace,
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: &deployer.Spec.Replicas,
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{"app": deployer.Name},
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{"app": deployer.Name},
                },
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{{
                        Name:  "main",
                        Image: deployer.Spec.Image,
                    }},
                },
            },
        },
    }
    // Apply Deployment
    if err := ctrl.SetControllerReference(deployer, deployment, r.Scheme); err != nil {
        return ctrl.Result{}, err
    }
    
    found := &appsv1.Deployment{}
    err := r.Get(ctx, types.NamespacedName{Name: deployer.Name, Namespace: deployer.Namespace}, found)
    if err != nil && errors.IsNotFound(err) {
        log.Info("Creating Deployment", "name", deployment.Name)
        if err := r.Create(ctx, deployment); err != nil {
            return ctrl.Result{}, err
        }
    } else if err == nil {
        // Update existing deployment if needed
        if !reflect.DeepEqual(found.Spec, deployment.Spec) {
            found.Spec = deployment.Spec
            log.Info("Updating Deployment", "name", deployment.Name)
            if err := r.Update(ctx, found); err != nil {
                return ctrl.Result{}, err
            }
        }
    }
    // Update status
    deployer.Status.Ready = (found.Status.ReadyReplicas == *found.Spec.Replicas)
    if err := r.Status().Update(ctx, deployer); err != nil {
        return ctrl.Result{}, err
    }

	return ctrl.Result{}, nil
}

// Helper functions
func setCondition(aim *aciv1alpha1.AIM, condType string, status metav1.ConditionStatus, reason, message string) {
    meta.SetStatusCondition(&aim.Status.Conditions, metav1.Condition{
        Type:    condType,
        Status:  status,
        Reason:  reason,
        Message: message,
    })
}
func (r *AIMReconciler) updateStatusAndRequeue(ctx context.Context, aim *aciv1alpha1.AIM) (ctrl.Result, error) {
    _ = r.Status().Update(ctx, aim)
    return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CiscoAciAimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ciscoaciaimv1.CiscoAciAim{}).
		Complete(r)
}
