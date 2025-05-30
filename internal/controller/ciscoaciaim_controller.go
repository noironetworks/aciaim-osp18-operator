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
	"reflect"
	"context"
	"time"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
//	mariadbv1 "github.com/openstack-k8s-operators/mariadb-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	ciscoaciaimv1 "github.com/noironetworks/aciaim-osp18-operator/api/v1alpha1"
//	rabbitmqv1 "github.com/openstack-k8s-operators/infra-operator/apis/rabbitmq/v1beta1"
)

// CiscoAciAimReconciler reconciles a CiscoAciAim object
type CiscoAciAimReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var kollaConfigJSON = `{
{
  "command": "/bin/supervisord -c /etc/aim/aim_supervisord.conf",
  "config_files": [
    {
      "source": "/var/lib/kolla/config_files/src/*",
      "dest": "/",
      "merge": true,
      "preserve_properties": true
    }
  ],
  "permissions": [
    {
      "path": "/var/log/aim",
      "owner": "neutron:neutron",
      "recurse": true
    }
  ]
}`

// GetLog returns a logger object with a prefix of "conroller.name" and aditional controller context fields
func (r *CiscoAciAimReconciler) GetLogger(ctx context.Context) logr.Logger {
	return log.FromContext(ctx).WithName("Controllers").WithName("CiscoAciAimController")
}

func (r *CiscoAciAimReconciler) updateStatusAndRequeue(ctx context.Context, aim *ciscoaciaimv1.CiscoAciAim) (ctrl.Result, error) {
    _ = r.Status().Update(ctx, aim)
    return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// +kubebuilder:rbac:groups=api.cisco.com,resources=ciscoaciaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=api.cisco.com,resources=ciscoaciaims/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=api.cisco.com,resources=ciscoaciaims/finalizers,verbs=update
// +kubebuilder:rbac:groups=api.cisco.com,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mariadb.openstack.org,resources=galeras,verbs=get;list;watch
// +kubebuilder:rbac:groups=rabbitmq.openstack.org,resources=rabbitmqclusters,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete;

    /*
    // --- MariaDB Readiness ---
    var mariadb mariadbv1.Galera
    if err := r.Get(ctx, types.NamespacedName{Name: instance.Spec.MariaDBInstance, Namespace: instance.Namespace}, &mariadb); err != nil {
        setCondition(instance, "MariaDBReady", metav1.ConditionFalse, "NotFound", "MariaDB instance not found")
        return r.updateStatusAndRequeue(ctx, instance)
    }
    if !meta.IsStatusConditionTrue(mariadb.Status.Conditions, mariadbv1.ConditionTypeReady) {
        setCondition(instance, "MariaDBReady", metav1.ConditionFalse, "NotReady", "MariaDB not ready")
        return r.updateStatusAndRequeue(ctx, instance)
    }

    // --- RabbitMQ Readiness ---
    var rabbitmq rabbitmqv1.RabbitmqCluster
    if err := r.Get(ctx, types.NamespacedName{Name: instance.Spec.RabbitMQCluster, Namespace: instance.Namespace}, &rabbitmq); err != nil {
        setCondition(instance, "RabbitMQReady", metav1.ConditionFalse, "NotFound", "RabbitMQ cluster not found")
        return r.updateStatusAndRequeue(ctx, instance)
    }
    if !rabbitmq.Status.Ready {
        setCondition(instance, "RabbitMQReady", metav1.ConditionFalse, "NotReady", "RabbitMQ not ready")
        return r.updateStatusAndRequeue(ctx, instance)
    }

    // --- Dependencies are ready, deploy AIM ---
    setCondition(instance, "MariaDBReady", metav1.ConditionTrue, "Ready", "MariaDB is ready")
    setCondition(instance, "RabbitMQReady", metav1.ConditionTrue, "Ready", "RabbitMQ is ready")
    if !meta.IsStatusConditionTrue(instance.Status.Conditions, "Ready") {
    if err := r.Status().Update(ctx, instance); err != nil {
        return ctrl.Result{}, err
    }
   }


    replicas := int32(2)
    if instance.Spec.Replicas != nil {
        replicas = *instance.Spec.Replicas
    }

    // Define Deployment
    deployment := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      instance.Name,
            Namespace: instance.Namespace,
            Labels:    map[string]string{"app": instance.Name},
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: &replicas,
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{"app": instance.Name},
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{"app": instance.Name},
                },
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{{
                        Name:  "aim",
                        Image: instance.Spec.ContainerImage,
                   }},
                },
            },
        },
    }
    // Set owner reference for garbage collection
    if err := ctrl.SetControllerReference(instance, deployment, r.Scheme); err != nil {
        return ctrl.Result{}, err
    }

    found := &appsv1.Deployment{}
    err = r.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, found)
    if err != nil && k8s_errors.IsNotFound(err) {
        Log.Info("Creating Deployment", "name", deployment.Name)
        if err := r.Create(ctx, deployment); err != nil {
            return ctrl.Result{}, err
        }
    } else if err == nil {
        // Update existing deployment if needed
        if !reflect.DeepEqual(found.Spec, deployment.Spec) {
            found.Spec = deployment.Spec
            Log.Info("Updating Deployment", "name", deployment.Name)
            if err := r.Update(ctx, found); err != nil {
                return ctrl.Result{}, err
            }
        }
    }


    // Update status using conditions
    readyCondition := metav1.Condition{
        Type:    "Ready",
        Status:  metav1.ConditionFalse,
        Reason:  "Progressing",
        Message: "Deployment is progressing",
    }

    if found.Status.ReadyReplicas == *found.Spec.Replicas {
        readyCondition.Status = metav1.ConditionTrue
        readyCondition.Reason = "AllReplicasReady"
        readyCondition.Message = "All replicas are ready"
    }

    meta.SetStatusCondition(&instance.Status.Conditions, readyCondition)

    if err := r.Status().Update(ctx, instance); err != nil {
        return ctrl.Result{}, err
    }


	return ctrl.Result{}, nil
}
*/


func (r *CiscoAciAimReconciler) ensureConfigMap(ctx context.Context, instance *ciscoaciaimv1.CiscoAciAim) (*corev1.ConfigMap, error) {
    configMapName := instance.Name + "-config"
    Log := r.GetLogger(ctx)

    // Create the ConfigMap with both config files and the Kolla config.json
    configMap := &corev1.ConfigMap{
        ObjectMeta: metav1.ObjectMeta{
            Name:      configMapName,
            Namespace: instance.Namespace,
            Labels:    map[string]string{"app": instance.Name},
        },
        Data: map[string]string{
//            "aim.conf":    "[DEFAULT]\ndebug = true\n", // Your main config
  //          "aimctl.conf": "[DEFAULT]\n# aimctl config\n", // Static config
            "config.json": kollaConfigJSON, // Kolla config - THIS IS THE KEY LINE
        },
    }

    // Set owner reference so ConfigMap is deleted when CR is deleted
    if err := ctrl.SetControllerReference(instance, configMap, r.Scheme); err != nil {
        return nil, err
    }

    // Check if ConfigMap already exists
    found := &corev1.ConfigMap{}
    err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: instance.Namespace}, found)
    if err != nil && k8s_errors.IsNotFound(err) {
        Log.Info("Creating ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
        err = r.Create(ctx, configMap)
        if err != nil {
            return nil, err
        }
    } else if err != nil {
        return nil, err
    } else {
        // Update existing ConfigMap if content has changed
        if !reflect.DeepEqual(found.Data, configMap.Data) {
            found.Data = configMap.Data
            Log.Info("Updating ConfigMap", "ConfigMap.Namespace", found.Namespace, "ConfigMap.Name", found.Name)
            err = r.Update(ctx, found)
            if err != nil {
                return nil, err
            }
        }
    }

    return configMap, nil
}

func (r *CiscoAciAimReconciler) ensureDeployment(ctx context.Context, instance *ciscoaciaimv1.CiscoAciAim, configMap *corev1.ConfigMap) error {
    Log := r.GetLogger(ctx)
    deploymentName := instance.Name + "-deployment"
    replicas := int32(2)                                                           
    if instance.Spec.Replicas != nil {                                             
        replicas = *instance.Spec.Replicas                                         
    }
    // Define volume mounts for the container
    volumeMounts := []corev1.VolumeMount{
        {
            Name:      "config-volume",
            MountPath: "/var/lib/kolla/config_files", // This is crucial!
            ReadOnly:  true,
        },
    }

    // Define volumes for the pod
    volumes := []corev1.Volume{
        {
            Name: "config-volume",
            VolumeSource: corev1.VolumeSource{
                ConfigMap: &corev1.ConfigMapVolumeSource{
                    LocalObjectReference: corev1.LocalObjectReference{
                        Name: configMap.Name,
                    },
                },
            },
        },
    }

    envVars := map[string]env.Setter{}
	envVars["KOLLA_CONFIG_STRATEGY"] = env.SetValue("COPY_ALWAYS")
	envVars["KOLLA_BOOTSTRAP"] = env.SetValue("true")

    trueVal := true
    // Create the deployment spec
    deployment := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      instance.Name,
            Namespace: instance.Namespace,
            Labels:    map[string]string{"app": instance.Name},
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: &replicas,
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{"app": instance.Name},
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{"app": instance.Name},
                },
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{
                        {
                            Name:  "aciaim",
                            Image: instance.Spec.ContainerImage,
     //                       Ports: []corev1.ContainerPort{
       //                         {
         //                           ContainerPort: 8080,
           //                         Name:          "http",
             //                   },
               //             },
                            VolumeMounts: volumeMounts, // Mount the volumes
                            // Ensure the container runs kolla_set_configs first
                            Env: env.MergeEnvs([]corev1.EnvVar{}, envVars),
                            Command: []string{"/bin/bash"},
                            Args: []string{
                                "-c",
                                "sudo kolla_set_configs && exec /usr/bin/aciaim-server",
                            },
                            SecurityContext: &corev1.SecurityContext{
                                Privileged: &trueVal,// privileged: true
                            },
                            LivenessProbe: &corev1.Probe{
                                ProbeHandler: corev1.ProbeHandler{
                                    Exec: &corev1.ExecAction{
                                        Command: []string{"/etc/aim/aim_healthcheck"},
                                    },
                                },
                                InitialDelaySeconds: 30,
                                PeriodSeconds:       10,
                            },
                        },
                    },
                    Volumes: volumes, // Add the volumes to pod spec
                    HostNetwork: true, // net: host
                    HostPID:     true, // pid: host
                },
            },
        },
    }

    // Set owner reference
    if err := ctrl.SetControllerReference(instance, deployment, r.Scheme); err != nil {
        return err
    }

    // Create or update deployment
    found := &appsv1.Deployment{}
    err := r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: instance.Namespace}, found)
    if err != nil && k8s_errors.IsNotFound(err) {
        Log.Info("Creating Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
        err = r.Create(ctx, deployment)
        if err != nil {
            return err
        }
    } else if err != nil {
        return err
    }

    return nil
}



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
    // Ensure ConfigMap exists
    configMap, err := r.ensureConfigMap(ctx, instance)
    if err != nil {
        Log.Error(err, "Failed to ensure ConfigMap")
        return ctrl.Result{}, err
    }

    // Ensure Deployment exists
    err = r.ensureDeployment(ctx, instance, configMap)
    if err != nil {
        Log.Error(err, "Failed to ensure Deployment")
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}

// Helper functions
func setCondition(aim *ciscoaciaimv1.CiscoAciAim, condType string, status metav1.ConditionStatus, reason, message string) {
    meta.SetStatusCondition(&aim.Status.Conditions, metav1.Condition{
        Type:    condType,
        Status:  status,
        Reason:  reason,
        Message: message,
    })
}

// SetupWithManager sets up the controller with the Manager.
func (r *CiscoAciAimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ciscoaciaimv1.CiscoAciAim{}).
		Complete(r)
}
