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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	ciscoaciaimv1 "github.com/noironetworks/aciaim-osp18-operator/api/v1alpha1"
	aciaim "github.com/noironetworks/aciaim-osp18-operator/pkg/ciscoaciaim"
	rabbitmqv1 "github.com/openstack-k8s-operators/infra-operator/apis/rabbitmq/v1beta1"
	mariadbv1 "github.com/openstack-k8s-operators/mariadb-operator/api/v1beta1"
	neutronv1 "github.com/openstack-k8s-operators/neutron-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"path/filepath"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"text/template"
	"time"
)

const templatePath = "/templates"

// CiscoAciAimReconciler reconciles a CiscoAciAim object
type CiscoAciAimReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type PhysDomConfig struct {
	PhysDomName string
}

type AimConfData struct {
	DatabaseConnection   string
	MessageBusConnection string
	// Top-level fields
	ACIAimDebug                bool
	ACIScopeNames              bool
	ACIGen1HwGratArps          bool
	ACIEnableFaultSubscription bool

	// AciConnection fields
	ACIApicHosts    string
	ACIApicUsername string
	ACIApicPassword string
	ACIApicCertName string
	ACIApicSystemId string
	AgentIDBase     string
	ACIVerifySslCertificate  string
}

type AimCtlConfData struct {
	// AciConnection fields
	ACIApicSystemId          string
	ACIApicSystemIdMaxLength int

	// AciFabric fields
	ACIOpflexEncapMode      string
	AciVmmMcastRanges       string
	AciVmmMulticastAddress  string
	ACIOpflexVlanRange      string // Comma-separated
	ACIApicEntityProfile    string
	AciExternalRoutedDomain string
	ACIVpcPairs             string // Comma-separated

	// Top-level field
	ACIScopeInfra bool

	// Processed data for template loops
	ACIHostLinksMap map[string]map[string]string
	PhysDomConfigs  []PhysDomConfig
}

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
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps;secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch
// +kubebuilder:rbac:groups=rabbitmq.openstack.org,resources=transporturls,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

func (r *CiscoAciAimReconciler) ensureConfigMap(ctx context.Context, instance *ciscoaciaimv1.CiscoAciAim, configData map[string]string) (*corev1.ConfigMap, error) {
	Log := r.GetLogger(ctx)

	// Create the ConfigMap object using the builder from pkg/ciscoaciaim
	configMap := aciaim.ConfigMap(instance, configData)

	// Set owner reference so ConfigMap is deleted when CR is deleted
	if err := ctrl.SetControllerReference(instance, configMap, r.Scheme); err != nil {
		return nil, err
	}

	// Check if ConfigMap already exists
	found := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: configMap.Name, Namespace: instance.Namespace}, found)
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

func (r *CiscoAciAimReconciler) ensureLogPVC(ctx context.Context,
	instance *ciscoaciaimv1.CiscoAciAim) error {
	Log := r.GetLogger(ctx)
	pvcName := instance.Name + "-log-pvc"

	// Create the PVC object using the builder from pkg/ciscoaciaim
	newPvc, err := aciaim.LogPVC(instance)
	if err != nil {
		return fmt.Errorf("failed to create PVC object: %w", err)
	}

	// Set the CR as the owner of the PVC.
	if err := ctrl.SetControllerReference(instance, newPvc, r.Scheme); err != nil {
		return err
	}

	found := &corev1.PersistentVolumeClaim{}
	err = r.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: instance.Namespace}, found)
	if err != nil && k8s_errors.IsNotFound(err) {
		Log.Info("Creating a new PersistentVolumeClaim for logs", "PVC.Name", pvcName)
		return r.Create(ctx, newPvc)
	} else if err != nil {
		// Another error occurred while trying to get the PVC.
		return err
	}

	// If PVC exists, update only mutable fields (e.g., storage requests)
	// Do NOT update StorageClassName or other immutable fields
	if !found.Spec.Resources.Requests.Storage().Equal(*newPvc.Spec.Resources.Requests.Storage()) ||
		!reflect.DeepEqual(found.ObjectMeta.Labels, newPvc.ObjectMeta.Labels) {

		Log.Info("Updating existing PersistentVolumeClaim for logs", "PVC.Name", pvcName)
		// Update only the storage request
		found.Spec.Resources.Requests = newPvc.Spec.Resources.Requests
		found.ObjectMeta.Labels = newPvc.ObjectMeta.Labels
		err = r.Update(ctx, found)
		if err != nil {
			return fmt.Errorf("failed to update PVC %s: %w", pvcName, err)
		}
	} else {
		Log.Info("Log PVC already exists and is up-to-date.", "PVC.Name", pvcName)
	}

	return nil
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

	setCondition(instance, aciaim.ConditionDBReady, metav1.ConditionTrue, "DBReady", "Database connection is ready")

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

	setCondition(instance, aciaim.ConditionRabbitMQReady, metav1.ConditionTrue, "RabbitMQReady", "RabbitMQ transport URL is ready")

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

func (r *CiscoAciAimReconciler) populateAimConfData(
	ctx context.Context,
	dbConn string,
	busConn string,
	instance *ciscoaciaimv1.CiscoAciAim,
) (*AimConfData, error) {
	data := &AimConfData{
		DatabaseConnection:         dbConn,
		MessageBusConnection:       busConn,
		ACIAimDebug:                instance.Spec.ACIAimDebug,
		ACIScopeNames:              instance.Spec.ACIScopeNames,
		ACIGen1HwGratArps:          instance.Spec.AciGen1HwGratArps,
		ACIEnableFaultSubscription: instance.Spec.AciEnableFaultSubscription,
		ACIApicHosts:               instance.Spec.AciConnection.ACIApicHosts,
		ACIApicUsername:            instance.Spec.AciConnection.ACIApicUsername,
		ACIApicPassword:            instance.Spec.AciConnection.ACIApicPassword,
		ACIApicCertName:            instance.Spec.AciConnection.ACIApicCertName,
		ACIApicSystemId:            instance.Spec.AciConnection.ACIApicSystemId,
		AgentIDBase:                instance.Name,
		ACIVerifySslCertificate:    instance.Spec.AciConnection.ACIVerifySslCertificate,
	}

	return data, nil
}
func (r *CiscoAciAimReconciler) populateAimCtlConfData(
	ctx context.Context,
	instance *ciscoaciaimv1.CiscoAciAim,
) (*AimCtlConfData, error) {
	log := log.FromContext(ctx)

	data := &AimCtlConfData{
		ACIApicSystemId:          instance.Spec.AciConnection.ACIApicSystemId,
		ACIApicSystemIdMaxLength: instance.Spec.AciConnection.ACIApicSystemIdMaxLength,
		ACIScopeInfra:            instance.Spec.ACIScopeInfra,
	}

	if instance.Spec.AciFabric != nil {
		log.Info("AciFabric spec provided, populating aimctl.conf fabric configurations.")
		fab := instance.Spec.AciFabric

		data.ACIOpflexEncapMode = fab.ACIOpflexEncapMode
		data.AciVmmMcastRanges = fab.AciVmmMcastRanges
		data.AciVmmMulticastAddress = fab.AciVmmMulticastAddress
		data.ACIApicEntityProfile = fab.ACIApicEntityProfile
		data.AciExternalRoutedDomain = fab.AciExternalRoutedDomain
		data.ACIVpcPairs = strings.Join(fab.ACIVpcPairs, ",")
		data.ACIOpflexVlanRange = strings.Join(fab.ACIOpflexVlanRange, ",")

		if fab.ACIHostLinks != nil && len(fab.ACIHostLinks.Raw) > 0 {
			if err := json.Unmarshal(fab.ACIHostLinks.Raw, &data.ACIHostLinksMap); err != nil {
				return nil, fmt.Errorf("failed to parse ACIHostLinks JSON: %w", err)
			}
		}

		pdomMapping := make(map[string]string)
		for _, mappingStr := range fab.AciPhysDomMappings {
			parts := strings.SplitN(mappingStr, ":", 2)
			if len(parts) == 2 {
				pdomMapping[parts[0]] = parts[1]
			}
		}
		physDomSet := make(map[string]bool)
		for _, vlanRangeStr := range fab.NeutronNetworkVLANRanges {
			physNetName := strings.SplitN(vlanRangeStr, ":", 2)[0]
			var physDomName string
			if customPdom, ok := pdomMapping[physNetName]; ok {
				physDomName = customPdom
			} else {
				physDomName = fmt.Sprintf("pdom_%s", physNetName)
			}
			if !physDomSet[physDomName] {
				data.PhysDomConfigs = append(data.PhysDomConfigs, PhysDomConfig{PhysDomName: physDomName})
				physDomSet[physDomName] = true
			}
		}
	}

	return data, nil
}

func (r *CiscoAciAimReconciler) ensureStatefulSet(ctx context.Context, instance *ciscoaciaimv1.CiscoAciAim, configMap *corev1.ConfigMap, configMapChecksum string) error {
	Log := r.GetLogger(ctx)
	statefulSetName := instance.Name
	pvcName := instance.Name + "-log-pvc"

	// Create the StatefulSet object using the builder from pkg/ciscoaciaim
	statefulSet := aciaim.StatefulSet(instance, configMap.Name, pvcName, configMapChecksum) // CALL NEW StatefulSet FUNCTION

	// Set owner reference
	if err := ctrl.SetControllerReference(instance, statefulSet, r.Scheme); err != nil {
		return err
	}

	// Create or update StatefulSet
	found := &appsv1.StatefulSet{}
	err := r.Get(ctx, types.NamespacedName{Name: statefulSetName, Namespace: instance.Namespace}, found)
	if err != nil && k8s_errors.IsNotFound(err) {
		Log.Info("Creating StatefulSet", "StatefulSet.Namespace", statefulSet.Namespace, "StatefulSet.Name", statefulSet.Name)
		err = r.Create(ctx, statefulSet)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		// If StatefulSet exists, check if it needs to be updated
		if !reflect.DeepEqual(found.Spec, statefulSet.Spec) || !reflect.DeepEqual(found.ObjectMeta.Labels, statefulSet.ObjectMeta.Labels) {
			Log.Info("Updating StatefulSet", "StatefulSet.Namespace", found.Namespace, "StatefulSet.Name", found.Name)
			found.Spec = statefulSet.Spec
			found.ObjectMeta.Labels = statefulSet.ObjectMeta.Labels
			err = r.Update(ctx, found)
			if err != nil {
				return err
			}
		} else {
			Log.Info("StatefulSet already exists and is up-to-date.", "StatefulSet.Name", statefulSetName)
		}
	}

	return nil
}

func (r *CiscoAciAimReconciler) getTemplateContent(ctx context.Context, templateName string) ([]byte, error) {
	Log := r.GetLogger(ctx)
	fullPath := filepath.Join(templatePath, templateName)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		Log.Error(err, "Failed to read template file from filesystem", "path", fullPath)
		return nil, err
	}

	return content, nil
}

func (r *CiscoAciAimReconciler) generateConfigFiles(ctx context.Context, instance *ciscoaciaimv1.CiscoAciAim, dbConn, busConn string) (map[string]string, error) {
	configFiles := make(map[string]string)

	// Populate data structs
	aimConfData, err := r.populateAimConfData(ctx, dbConn, busConn, instance)
	if err != nil {
		return nil, err
	}

	aimCtlConfData, err := r.populateAimCtlConfData(ctx, instance)
	if err != nil {
		return nil, err
	}

	// Helper to execute templates
	executeTemplate := func(name, tmplContent string, data interface{}) (string, error) {
		var buf bytes.Buffer
		t, err := template.New(name).Parse(tmplContent)
		if err != nil {
			return "", fmt.Errorf("failed to parse template %s: %w", name, err)
		}
		if err := t.Execute(&buf, data); err != nil {
			return "", fmt.Errorf("failed to execute template %s: %w", name, err)
		}
		return buf.String(), nil
	}

	aimConfTmplContent, err := r.getTemplateContent(ctx, "aim.conf.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to read aim.conf.tmpl: %w", err)
	}

	aimCtlConfTmplContent, err := r.getTemplateContent(ctx, "aimctl.conf.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to read aimctl.conf.tmpl: %w", err)
	}

	// Generate file contents
	configFiles["aim.conf"], err = executeTemplate("aim.conf", string(aimConfTmplContent), aimConfData)
	if err != nil {
		return nil, err
	}

	configFiles["aimctl.conf"], err = executeTemplate("aimctl.conf", string(aimCtlConfTmplContent), aimCtlConfData)
	if err != nil {
		return nil, err
	}

	// Read static file contents from embedded filesystem
	kollaConfigJSONContent, err := r.getTemplateContent(ctx, "kolla_config.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read kolla_config.json: %w", err)
	}
	configFiles["kolla_config.json"] = string(kollaConfigJSONContent)

	healthcheckContent, err := r.getTemplateContent(ctx, "aim_healthcheck.sh")
	if err != nil {
		return nil, fmt.Errorf("failed to read aim_healthcheck.sh: %w", err)
	}
	configFiles["aim_healthcheck.sh"] = string(healthcheckContent)

	supervisordConfContent, err := r.getTemplateContent(ctx, "aim_supervisord.conf")
	if err != nil {
		return nil, fmt.Errorf("failed to read aim_supervisord.conf: %w", err)
	}
	configFiles["aim_supervisord.conf"] = string(supervisordConfContent)

	initScriptTemplateContent, err := r.getTemplateContent(ctx, "init.sh")
	if err != nil {
		return nil, fmt.Errorf("failed to read init.sh: %w", err)
	}
	configFiles["init.sh"] = string(initScriptTemplateContent)

	return configFiles, nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
// Reconcile is the main operator logic loop.
func (r *CiscoAciAimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)

	// 1. Fetch the CiscoAciAim instance
	instance := &ciscoaciaimv1.CiscoAciAim{} // Use the correct alias
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			Log.Info("CiscoAciAim resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		Log.Error(err, "Failed to get CiscoAciAim instance")
		return ctrl.Result{}, err
	}

	if err := r.ensureLogPVC(ctx, instance); err != nil {
		Log.Error(err, "Failed to ensure log PVC")
		return ctrl.Result{}, err
	}

	dbConn, err := r.ensureDB(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}
	if dbConn == "" {
		Log.Info("Database connection secret not ready, requeueing.")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	busConn, err := r.ensureMessageBus(ctx, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	// 3. Generate all configuration files using the dedicated helper function
	Log.Info("Generating configuration files...")
	configFiles, err := r.generateConfigFiles(ctx, instance, dbConn, busConn)
	if err != nil {
		Log.Error(err, "Failed to generate configuration files")
		return ctrl.Result{}, err
	}
	//Checksum
	configMapChecksum := aciaim.GenerateConfigMapChecksum(configFiles)

	// 4. Ensure the ConfigMap with all files exists and is up-to-date
	Log.Info("Ensuring ConfigMap is up-to-date...")
	configMap, err := r.ensureConfigMap(ctx, instance, configFiles)
	if err != nil {
		Log.Error(err, "Failed to ensure ConfigMap")
		return ctrl.Result{}, err
	}

	// 5. Ensure other dependent resources like secrets are present
	Log.Info("Ensuring RabbitMQ CA Secret...")
	err = r.ensureRabbitMQCaSecret(ctx, instance)
	if err != nil {
		Log.Error(err, "Failed to ensure RabbitMQ CA Secret")
		return ctrl.Result{}, err
	}

	// 6. Reconcile the main StatefulSet for the AIM service
	Log.Info("Ensuring StateFulSet is up-to-date...")
	err = r.ensureStatefulSet(ctx, instance, configMap, configMapChecksum)
	if err != nil {
		Log.Error(err, "Failed to ensure Deployment")
		return ctrl.Result{}, err
	}

	// If we reach here, deployment is successful
	setCondition(instance, aciaim.ConditionAimReady, metav1.ConditionTrue, "DeploymentReady", "Aim deployment is ready")

	// Update status conditions on the CR
	err = r.Status().Update(ctx, instance)
	if err != nil {
		Log.Error(err, "Failed to update CiscoAciAim status conditions")
		return ctrl.Result{}, err
	}

	Log.Info("Reconciliation complete.")
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
		Owns(&appsv1.StatefulSet{}).
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
