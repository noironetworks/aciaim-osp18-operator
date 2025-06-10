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

package v1alpha1

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
    pointer "k8s.io/utils/pointer"
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

const (
	// neutron:neutron
	NeutronUID int64 = 42435
	NeutronGID int64 = 42435
)

const (
	ServiceCommand = "/usr/local/bin/kolla_start"
)

var kollaConfigJSON = `
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

var aimConf = `
[DEFAULT]
debug=False
rpc_backend=rabbit
control_exchange=neutron
default_log_levels=neutron.context=ERROR
logging_default_format_string="%(asctime)s.%(msecs)03d %(process)d %(thread)d %(levelname)s %(name)s [-] %(instance)s%(message)s"
transport_url=rabbit://guest:w6596qmezS0waTFmGwA8uAnh4@overcloud-controller-0.internalapi.localdomain:5672/?ssl=0

[oslo_messaging_rabbit]
#rabbit_host=<rabbit-mq-host>
#rabbit_port=<rabbit-m1-port>
#rabbit_hosts=<rabbit-mq-host>:<rabbit-m1-port>
rabbit_use_ssl=False
#rabbit_userid=username
#rabbit_password=password
rabbit_ha_queues=False

[database]
# Example:
connection=mysql+pymysql://neutron_3893:b15c843ea92bbb01da8f278b05ff4666@openstack.openstack.svc/neutron?read_default_file=/etc/my.cnf
# Replace 127.0.0.1 above with the IP address of the database used by the
# main neutron server. (Leave it as is if the database runs on this host.)
# connection = sqlite://
# NOTE: In deployment the [database] section and its connection attribute may
# be set in the corresponding core plugin '.ini' file. However, it is suggested
# to put the [database] section and its connection attribute in this
# configuration file.

# Database engine for which script will be generated when using offline
# migration
# engine =

[aim]
# Seconds to regard the agent is down; should be at least twice report_interval.
# agent_down_time = 75
agent_down_time = 75
poll_config = False
aim_system_id = ostack-pt-1
support_gen1_hw_gratarps=False
enable_faults_subscriptions=False

[apic]
# Hostname:port list of APIC controllers
#apic_hosts = <ip:port>,<ip:port>,<ip:port>
apic_hosts=10.30.120.190
# Username for the APIC controller
#apic_username = <username>
apic_username=admin
# Password for the APIC controller
#apic_password = <password>
apic_password=noir0123
# Whether use SSl for connecting to the APIC controller or not
#apic_use_ssl = <flag>
apic_use_ssl=True
verify_ssl_certificate=False
scope_names=True
`
var aimctlConf = `
[DEFAULT]
apic_system_id=ostack-pt-1
apic_system_id_length=16


[apic]
# Note: When deploying multiple clouds against one APIC,
#       these names must be unique between the clouds.
# apic_vmm_type = OpenStack
# apic_vlan_ns_name = openstack_ns
# apic_node_profile = openstack_profile
# apic_entity_profile = openstack_entity
apic_entity_profile=sauto_ostack-pt-1_aep
# apic_function_profile = openstack_function

# Specify your network topology.
# This section indicates how your compute nodes are connected to the fabric's
# switches and ports. The format is as follows:
#
# [apic_switch:<swich_id_from_the_apic>]
# <compute_host>,<compute_host> = <switchport_the_host(s)_are_connected_to>
#
# You can have multiple sections, one for each switch in your fabric that is
# participating in Openstack. e.g.
#
# [apic_switch:17]
# ubuntu,ubuntu1 = 1/10
# ubuntu2,ubuntu3 = 1/11
#
# [apic_switch:18]
# ubuntu5,ubuntu6 = 1/1
# ubuntu7,ubuntu8 = 1/2
# APIC domains are specified by the following sections:
# [apic_physdom:<name>]
#
# [apic_vmdom:<name>]
#
# In the above sections, [apic] configurations can be overridden for more granular infrastructure sharing.
# What is configured in the [apic] sharing will be the default used in case a more specific configuration is missing
# for the domain.
# An example follows:
#
# [apic_vmdom:openstack_domain]
# vlan_ranges=1000:2000
#
# [apic_vmdom:openstack_domain_2]
# vlan_ranges=3000:4000
scope_infra=False
apic_provision_infra=False
apic_provision_hostlinks=False
apic_vpc_pairs=101:102

[apic_vmdom:ostack-pt-1]
`
var healthcheck = `#!/bin/sh

aim_aid_status=$(supervisorctl -c /etc/aim/aim_supervisord.conf status aim-aid | awk -F ' ' '{print $2}')
aim_event_status=$(supervisorctl -c /etc/aim/aim_supervisord.conf status aim-event | awk -F ' ' '{print $2}')
aim_rpc_status=$(supervisorctl -c /etc/aim/aim_supervisord.conf status aim-rpc | awk -F ' ' '{print $2}')

if [ "$aim_aid_status" != "RUNNING" ] || [ "$aim_event_status" != "RUNNING" ] || [ "$aim_rpc_status" != "RUNNING" ]; then
   echo "fail"
   exit 1
fi
`

var supervisordConf = `
[rpcinterface:supervisor]
supervisor.rpcinterface_factory = supervisor.rpcinterface:make_main_rpcinterface

[unix_http_server]
file = /tmp/aim-supervisord.sock

[supervisorctl]
serverurl = unix:///tmp/aim-supervisord.sock
prompt = ciscoaci-aim

[supervisord]
identifier = aim-supervisor
pidfile = /run/aid/aim-supervisor.pid
logfile = /var/log/aim/aim-supervisor.log
logfile_maxbytes = 10MB
logfile_backups = 3
loglevel = debug
childlogdir = /var/log/aim
umask = 022
minfds = 1024
minprocs = 200
nodaemon = true
nocleanup = false
strip_ansi = false

[program:aim-aid]
command=/usr/bin/aim-aid --config-file=/etc/aim/aim.conf --log-file=/var/log/aim/aim-aid.log
exitcodes=0,2
stopasgroup=true
startsecs=0
startretries=3
stopwaitsecs=10
autorestart=true
stdout_logfile=NONE
stderr_logfile=NONE

[program:aim-event]
command=/usr/bin/aim-event-service-polling --config-file=/etc/aim/aim.conf --log-file=/var/log/aim/aim-event-service-polling.log
exitcodes=0,2
stopasgroup=true
startsecs=10
startretries=3
stopwaitsecs=10
autorestart=true
stdout_logfile=NONE
stderr_logfile=NONE

[program:aim-rpc]
command=/usr/bin/aim-event-service-rpc --config-file=/etc/aim/aim.conf --log-file=/var/log/aim/aim-event-service-rpc.log
exitcodes=0,2
stopasgroup=true
startsecs=10
startretries=3
stopwaitsecs=10
autorestart=true
stdout_logfile=NONE
stderr_logfile=NONE
`

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
            "aim.conf": aimConf,
            "aimctl.conf": aimctlConf,
            "config.json": kollaConfigJSON, // Kolla config - THIS IS THE KEY LINE
            "aim_healthcheck": healthcheck,
            "aim_supervisord.conf": supervisordConf,
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
            MountPath: "/var/lib/kolla/config_files/src/etc/aim",
            ReadOnly:  true,
        },
        {
            Name:      "config-kolla",
            MountPath: "/var/lib/kolla/config_files/",
            ReadOnly:  true,
        },
/*
       {
            Name:      "lib-modules",
            MountPath: "/lib/modules",
            ReadOnly:  true,
        },*/
        {
            Name:      "aim-logs",
            MountPath: "/var/log/aim",
            ReadOnly:  false, // Needs write access for logs
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
                    Items: []corev1.KeyToPath{
                        {Key: "aim.conf", Path: "aim.conf"},
                        {Key: "aim_supervisord.conf", Path: "aim_supervisord.conf"},
                        {Key: "aim_healthcheck", Path: "aim_healthcheck"},
                        {Key: "aimctl.conf", Path: "aimctl.conf"},
                    },
                    DefaultMode: pointer.Int32(0755),
                },
            },
        },
        {
            Name: "config-kolla",
            VolumeSource: corev1.VolumeSource{
                ConfigMap: &corev1.ConfigMapVolumeSource{
                    LocalObjectReference: corev1.LocalObjectReference{
                        Name: configMap.Name,
                    },
                    Items: []corev1.KeyToPath{
                        {Key: "config.json", Path: "config.json"},
                    },
                },
            },
        },
/*
        {
            Name: "lib-modules",
            VolumeSource: corev1.VolumeSource{
                HostPath: &corev1.HostPathVolumeSource{
                    Path: "/lib/modules",
                    Type: &[]corev1.HostPathType{corev1.HostPathDirectory}[0],
                },
            },
        },*/
        {
            Name: "aim-logs",
            VolumeSource: corev1.VolumeSource{
                EmptyDir: &corev1.EmptyDirVolumeSource{},
            },
        },
    }

    envVars := map[string]env.Setter{}
	envVars["KOLLA_CONFIG_STRATEGY"] = env.SetValue("COPY_ALWAYS")
	envVars["KOLLA_BOOTSTRAP"] = env.SetValue("true")
    args := []string{"-c", ServiceCommand}
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
                    SecurityContext: &corev1.PodSecurityContext{
                        FSGroup:    pointer.Int64(NeutronUID),
                    },
                    ServiceAccountName: "neutron-neutron",
                    Containers: []corev1.Container{
                        {
                            Name:  "aciaim",
                            Image: instance.Spec.ContainerImage,
                            VolumeMounts: volumeMounts, // Mount the volumes
 							Command:                  []string{"/bin/bash"},
                            Args: args,
                            Env: env.MergeEnvs([]corev1.EnvVar{}, envVars),
                            SecurityContext: &corev1.SecurityContext{
                                RunAsUser:    pointer.Int64(NeutronUID),
                                RunAsGroup:   pointer.Int64(NeutronGID),
                                RunAsNonRoot: &trueVal,
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
