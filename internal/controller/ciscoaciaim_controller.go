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
	"bytes"
	"text/template"
	"fmt"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	pointer "k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	mariadbv1 "github.com/openstack-k8s-operators/mariadb-operator/api/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	neutronv1 "github.com/openstack-k8s-operators/neutron-operator/api/v1beta1"
	ciscoaciaimv1 "github.com/noironetworks/aciaim-osp18-operator/api/v1alpha1"
	rabbitmqv1 "github.com/openstack-k8s-operators/infra-operator/apis/rabbitmq/v1beta1"
)

// CiscoAciAimReconciler reconciles a CiscoAciAim object
type CiscoAciAimReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type AimConfData struct {
    DatabaseConnection string
    MessageBusConnection string
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

const aimConf = `
[DEFAULT]
debug=False
rpc_backend=rabbit
control_exchange=neutron
default_log_levels=neutron.context=ERROR
logging_default_format_string="%(asctime)s.%(msecs)03d %(process)d %(thread)d %(levelname)s %(name)s [-] %(instance)s%(message)s"
transport_url={{ .MessageBusConnection }}

[oslo_messaging_rabbit]
ssl=True
rabbit_ha_queues=False

[database]
# Example:
connection = {{ .DatabaseConnection }}
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
apic_hosts=10.30.120.148
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
apic_entity_profile=fab205-kube
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

[apic_vmdom:fab205]
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

// +kubebuilder:rbac:groups=api.cisco.com,resources=ciscoaciaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=api.cisco.com,resources=ciscoaciaims/status;ciscoaciaims/finalizers,verbs=get;update;patch
// +kubebuilder:rbac:groups=neutron.openstack.org,resources=neutronapis,verbs=get;list;watch
// +kubebuilder:rbac:groups=neutron.openstack.org,resources=neutronapis/status,verbs=get
// +kubebuilder:rbac:groups=mariadb.openstack.org,resources=mariadbaccounts,verbs=get;list;watch
// +kubebuilder:rbac:groups=mariadb.openstack.org,resources=mariadbdatabases,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps;secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch
// +kubebuilder:rbac:groups=rabbitmq.openstack.org,resources=transporturls,verbs=get;list;watch;create;update;patch;delete


func (r *CiscoAciAimReconciler) ensureConfigMap(ctx context.Context, instance *ciscoaciaimv1.CiscoAciAim, aimConfString string) (*corev1.ConfigMap, error) {
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
            "aim.conf": aimConfString,
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

func (r *CiscoAciAimReconciler) ensureDB(
    ctx context.Context,
    instance *ciscoaciaimv1.CiscoAciAim,
) (string, error) {
    Log := r.GetLogger(ctx)
    namespace := instance.Namespace

    neutronAPI_instance := &neutronv1.NeutronAPI{}
    err := r.Get(ctx, types.NamespacedName{Name: "neutron", Namespace: namespace}, neutronAPI_instance)
    if err != nil {
        return "", err
    }

    dbAccountName := neutronAPI_instance.Spec.DatabaseAccount

    db := &mariadbv1.MariaDBDatabase{}
    err = r.Get(ctx, types.NamespacedName{Name: dbAccountName, Namespace: namespace}, db)
    if err != nil {
        Log.Error(err, "Failed to get dbAccountName", "name", dbAccountName, "namespace", namespace)
        return "", err
    }

    dbAccount := &mariadbv1.MariaDBAccount{}
    err = r.Get(ctx, types.NamespacedName{Name: dbAccountName, Namespace: namespace}, dbAccount)
    if err != nil {
        Log.Error(err, "Failed to get MariaDBAccount", "name", dbAccountName, "namespace", namespace)
        return "", err
    }

    secretName := fmt.Sprintf("%s-db-secret", dbAccountName)
    secret := &corev1.Secret{}
    err = r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret)
    if err != nil {
        Log.Error(err, "Failed to get Secret for MariaDBAccount", "secretName", secretName, "namespace", namespace)
        return "", err
    }

    // Extract username and password
    dbUser := dbAccount.Spec.UserName
    dbPasswordBytes, ok := secret.Data[mariadbv1.DatabasePasswordSelector]
    if !ok {
        Log.Info("Password key missing in secret data", "secretName", secretName)
        return "", nil // or return error if preferred
    }
    dbPassword := string(dbPasswordBytes)
    dbHost := neutronAPI_instance.Status.DatabaseHostname
    dbName := "neutron"

    // Construct the connection string
    dbConn := fmt.Sprintf("mysql+pymysql://%s:%s@%s/%s?read_default_file=/etc/my.cnf",
        dbUser, dbPassword, dbHost, dbName)
    Log.Info("Constructed DB connection string", "dbConn", dbConn)

    return dbConn, nil
}

func (r *CiscoAciAimReconciler) ensureMessageBus(
    ctx context.Context,
    instance *ciscoaciaimv1.CiscoAciAim,
) (string, error) {
    Log := r.GetLogger(ctx)
    namespace := instance.Namespace

    neutronAPI_instance := &neutronv1.NeutronAPI{}
    err := r.Get(ctx, types.NamespacedName{Name: "neutron", Namespace: namespace}, neutronAPI_instance)
    if err != nil {
        return "", err
    }

    // Dynamically build the TransportURL resource name
    transportURLName := fmt.Sprintf("%s-neutron-transport", neutronAPI_instance.Name)
    transportURL := &rabbitmqv1.TransportURL{}
    err = r.Client.Get(ctx, client.ObjectKey{Name: transportURLName, Namespace: namespace}, transportURL)
    if err != nil {
        Log.Error(err, "Failed to get TransportURL resource", "name", transportURLName, "namespace", namespace)
        return "", err
    }

    secretName := "rabbitmq-transport-url-neutron-neutron-transport" // adjust if needed
    secret := &corev1.Secret{}
    err = r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret)
    if err != nil {
        Log.Error(err, "Failed to get RabbitMQ secret", "secretName", secretName, "namespace", namespace)
        return "", err
    }

    transportURLBytes, ok := secret.Data["transport_url"]
    if !ok {
        Log.Info("transport_url key missing in RabbitMQ secret", "secretName", secretName)
        return "", nil // or return error if preferred
    }

    transportURLStr := string(transportURLBytes)
    Log.Info("Retrieved RabbitMQ transport URL", "transportURL", transportURLStr)

    return transportURLStr, nil
}

func (r *CiscoAciAimReconciler) ensureRabbitMQCaSecret(ctx context.Context, instance *ciscoaciaimv1.CiscoAciAim) error {
    srcSecret := &corev1.Secret{}
    err := r.Get(ctx, types.NamespacedName{
        Name:      "rootca-internal",
        Namespace: instance.Namespace,
    }, srcSecret)
    if err != nil {
        return err
    }
    caCrt, ok := srcSecret.Data["ca.crt"]
    if !ok {
        return fmt.Errorf("ca.crt not found in rootca-internal secret")
    }
    // Prepare the new Secret object
    dstSecret := &corev1.Secret{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "rabbitmq-ca-secret",
            Namespace: instance.Namespace,
        },
        Type: corev1.SecretTypeOpaque,
        Data: map[string][]byte{
            "tls-ca-bundle.pem": caCrt,
        },
    }

    // Create or update as needed
    existing := &corev1.Secret{}
    err = r.Get(ctx, types.NamespacedName{
        Name:      "rabbitmq-ca-secret",
        Namespace: instance.Namespace,
    }, existing)
    if err != nil && k8s_errors.IsNotFound(err) {
        return r.Create(ctx, dstSecret)
    } else if err == nil {
        if !reflect.DeepEqual(existing.Data, dstSecret.Data) {
            existing.Data = dstSecret.Data
            return r.Update(ctx, existing)
        }
        return nil
    } else {
        return err
    }
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
        {
            Name:      "rabbitmq-ca",
            MountPath: "/etc/pki/ca-trust/extracted/pem/",
            ReadOnly:  true,
        },
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
        {
            Name: "rabbitmq-ca",
            VolumeSource: corev1.VolumeSource{
                Secret: &corev1.SecretVolumeSource{
                    SecretName: "rabbitmq-ca-secret",
                },
            },
        },
        {
            Name: "aim-logs",
            VolumeSource: corev1.VolumeSource{
                EmptyDir: &corev1.EmptyDirVolumeSource{},
            },
        },
    }

    envVars := map[string]env.Setter{}
	envVars["KOLLA_CONFIG_STRATEGY"] = env.SetValue("COPY_ALWAYS")
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

    dbConn, err := r.ensureDB(ctx, instance)
    if err != nil {
        return ctrl.Result{}, err
    }
    if dbConn == "" {
        // Secret not ready, requeue after some time
        return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
    }

    busConn, err := r.ensureMessageBus(ctx, instance)
    if err != nil {
        return ctrl.Result{}, err
    }
    // Now use dbConn to populate your aim.conf
    aimConfData := AimConfData{
                        DatabaseConnection: dbConn,
                        MessageBusConnection: busConn,
                   }
    var buf bytes.Buffer
    tmpl, err := template.New("aimconf").Parse(aimConf)
    if err != nil {
        return ctrl.Result{}, err
    }
    err = tmpl.Execute(&buf, aimConfData)
    if err != nil {
        return ctrl.Result{}, err
    }
    aimConfString := buf.String()

    err = r.ensureRabbitMQCaSecret(ctx, instance)
    if err != nil {
        Log.Error(err, "Failed to ensure RabbitMQ CA Secret")
        // Requeue or fail as appropriate for your logic
        return ctrl.Result{}, err
    }

    // Ensure ConfigMap exists
    configMap, err := r.ensureConfigMap(ctx, instance, aimConfString)
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

func mapMariaDBSecretToCiscoAciAim(c client.Client) handler.MapFunc {
    return func(ctx context.Context, obj client.Object) []reconcile.Request {
        secret, ok := obj.(*corev1.Secret)
        if !ok {
            return nil
        }

        // Only trigger on the Neutron DB Secret.
        // Adjust name and namespace as needed for your environment.
        if secret.Name == "neutron-db-secret" && secret.Namespace == "openstack" {
            // List all CiscoAciAim CRs in this namespace
            var aimList ciscoaciaimv1.CiscoAciAimList
            if err := c.List(ctx, &aimList, &client.ListOptions{Namespace: "openstack"}); err != nil {
                return nil
            }
            var reqs []reconcile.Request
            for _, aim := range aimList.Items {
                reqs = append(reqs, reconcile.Request{
                    NamespacedName: client.ObjectKey{
                        Name:      aim.Name,
                        Namespace: aim.Namespace,
                    },
                })
            }
            return reqs
        }
        return nil
    }
}

func mapRabbitMQSecretToCiscoAciAim(c client.Client) handler.MapFunc {
    return func(ctx context.Context, obj client.Object) []reconcile.Request {
        secret, ok := obj.(*corev1.Secret)
        if !ok {
            return nil
        }

        if secret.Name == "rabbitmq-transport-url-neutron-neutron-transport" && secret.Namespace == "openstack" {
            var aimList ciscoaciaimv1.CiscoAciAimList
            if err := c.List(ctx, &aimList, &client.ListOptions{Namespace: "openstack"}); err != nil {
                return nil
            }
            var reqs []reconcile.Request
            for _, aim := range aimList.Items {
                reqs = append(reqs, reconcile.Request{
                    NamespacedName: client.ObjectKey{
                        Name:      aim.Name,
                        Namespace: aim.Namespace,
                    },
                })
            }
            return reqs
        }
        return nil
    }
}

// SetupWithManager sets up the controller with the Manager.
func (r *CiscoAciAimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ciscoaciaimv1.CiscoAciAim{}).
        Watches(
            &corev1.Secret{},
            handler.EnqueueRequestsFromMapFunc(
                mapMariaDBSecretToCiscoAciAim(mgr.GetClient()),
            ),
        ).
        Watches(
            &corev1.Secret{},
            handler.EnqueueRequestsFromMapFunc(
                mapRabbitMQSecretToCiscoAciAim(mgr.GetClient()),
            ),
        ).
		Complete(r)
}
