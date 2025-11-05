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
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
const (
	// ContainerImage - default fall-back container image for CiscoAciAim if associated env var not provided
	ContainerImage = "registry.connect.redhat.com/noiro/openstack-ciscoaci-aim:latest"
)

type LogPersistenceSpec struct {
	// The size of the persistent volume to claim, e.g., "10Gi".
	// Required if persistence is enabled.
	// +kubebuilder:validation:Optional
	Size string `json:"size,omitempty"`

	// The name of the StorageClass to use for the PVC.
	// If omitted, the cluster's default StorageClass will be used.
	// +kubebuilder:validation:Optional
	StorageClassName string `json:"storageClassName,omitempty"`
}

// LivenessProbeSpec defines the liveness probe parameters.
type LivenessProbeSpec struct {
	// Initial delay before the probe starts.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=30
	// +kubebuilder:validation:Minimum=0
	InitialDelaySeconds int32 `json:"initialDelaySeconds,omitempty"`

	// How often to perform the probe.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=10
	// +kubebuilder:validation:Minimum=1
	PeriodSeconds int32 `json:"periodSeconds,omitempty"`

	// Timeout for the probe.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=5
	// +kubebuilder:validation:Minimum=1
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty"`

	// Minimum consecutive successes for the probe to be considered successful.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	SuccessThreshold int32 `json:"successThreshold,omitempty"`

	// Minimum consecutive failures for the probe to be considered failed.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=3
	// +kubebuilder:validation:Minimum=1
	FailureThreshold int32 `json:"failureThreshold,omitempty"`

	// DisableProbe allows disabling the liveness probe.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	DisableProbe bool `json:"disableProbe,omitempty"`
}

// AciConnectionSpec defines all the parameters needed to connect to the ACI APIC.
type AciConnectionSpec struct {
	// APIC ip address.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1.1.1.1"
	ACIApicHosts string `json:"ACIApicHosts,omitempty"`

	// Username for the APIC controller.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="admin"
	ACIApicUsername string `json:"ACIApicUsername,omitempty"`

	// A reference to the Kubernetes Secret containing the APIC password.
	// +kubebuilder:validation:Optional
	ACIApicPassword string `json:"ACIApicPassword,omitempty"`

	// ACIApicPasswordSecretRef *corev1.SecretKeySelector `json:"ACIApicPasswordSecretRef,omitempty"`

	// Certificate name for APIC authentication.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	ACIApicCertName string `json:"ACIApicCertName,omitempty"`

	// A reference to the Kubernetes Secret containing the private key for the certificate.
	// +kubebuilder:validation:Optional
	ACIApicPrivateKeySecretRef *corev1.SecretKeySelector `json:"ACIApicPrivateKeySecretRef,omitempty"`

	// The System ID is used to support running multiple OpenStack
	// clouds on a single ACI fabric. Resources created in ACI are
	// annotated with the System ID to associate the resource with
	// a given OpenStack cloud. The System ID is also used in the
	// generation of some name strings used for ACI resources as
	// well. In most installations, the System ID must not exceed
	// the default maximum length of 16 characters. However, in
	// some special cases, it can be increased, which requires
	// also setting the ACIApicSystemIdMaxLength parameter.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="aci_openstack"
	ACIApicSystemId string `json:"ACIApicSystemId,omitempty"`

	// Maximmum length of the ACIApicSystemId. Please consult the
	// business unit before changing this value from the default.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=16
	ACIApicSystemIdMaxLength int `json:"ACIApicSystemIdMaxLength,omitempty"`
}

// NEW: AciFabricSpec defines the ACI fabric integration and topology settings.
type AciFabricSpec struct {
	// The default Attachable Entity Profile (AEP) to be used for all
	// host links, unless overridden in ACIHostLinks.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="aci-openstack-aep"
	ACIApicEntityProfile string `json:"ACIApicEntityProfile,omitempty"`

	// A list of vPC pairs on the fabric. Each string should be in
	// comma separated string of switch id's which form vpc pairs.
	// Example '101:102,103:104'
	// +kubebuilder:validation:Optional
	ACIVpcPairs []string `json:"ACIVpcPairs,omitempty"`

	// The encapsulation mode to be used for OpFlex traffic.
	// Can be either 'vlan' or 'vxlan'.
	// +kubebuilder:validation:Enum=vlan;vxlan
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="vxlan"
	ACIOpflexEncapMode string `json:"ACIOpflexEncapMode,omitempty"`

	// The range of VLANs to be used when ACIOpflexEncapMode is 'vlan'.
	// this option is ignored when encap mode is 'vxlan'.
	// +kubebuilder:validation:Optional
	ACIOpflexVlanRange []string `json:"ACIOpflexVlanRange,omitempty"`

	// Describes the host connections to switches in json format.
	// +kubebuilder:validation:Optional
	// +kubebuilder:pruning:PreserveUnknownFields
	ACIHostLinks *apiextensionsv1.JSON `json:"ACIHostLinks,omitempty"`

	// Whether to enable ACI integration for Open vSwitch.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	AciOpenvswitch bool `json:"AciOpenvswitch,omitempty"`

	// Multicast address ranges for the VMM domain.
	// This is treated as a single string.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="225.2.1.1:225.2.255.255"
	AciVmmMcastRanges string `json:"AciVmmMcastRanges,omitempty"`

	// The specific multicast address for the VMM domain.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="225.1.2.3"
	AciVmmMulticastAddress string `json:"AciVmmMulticastAddress,omitempty"`

	// VLAN ranges for Neutron provider networks. Used for hierarchical port binding.
	// This value will be plugged into ml2_type_vlan section of plugin.ini if it is not blank
	// example value datacentre:1000:2000
	// +kubebuilder:validation:Optional
	NeutronNetworkVLANRanges []string `json:"NeutronNetworkVLANRanges,omitempty"`

	// Mappings of physical domains in ACI to Neutron provider networks.
	// Each string should be in the format "physnet_name:aci_domain_name".
	// Example: ["physnet0:my_pdom0", "physnet1:my_pdom1"]
	// +kubebuilder:validation:Optional
	AciPhysDomMappings []string `json:"AciPhysDomMappings,omitempty"`

	// Mappings of Neutron provider networks to physical device interfaces.
	// List of <physical_network>:<ACI Physdom>
	// By default each physnet maps to a precreated ACI
	// physdom with pdom_<physnet_name>. For example
	// physnet0 will map to physdom named pdom_phynet0
	// This parameter allows user to override the mapping.
	// Example: "physnet0:my_pdom0, physnet1:my_pdom1"
	// +kubebuilder:validation:Optional
	NeutronPhysicalDevMappings []string `json:"NeutronPhysicalDevMappings,omitempty"`

	// The name of the L3Out/External Routed Domain to use for floating IPs
	// and other external connectivity.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=""
	AciExternalRoutedDomain string `json:"AciExternalRoutedDomain,omitempty"`
}

// CiscoAciAimSpec defines the desired state of AciAim
type CiscoAciAimSpec struct {
	// Important: Run "make" to regenerate code after modifying this file
	// +kubebuilder:validation:Required
	// Cisco Aci Aim Container Image URL
	ContainerImage string `json:"containerImage"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Maximum=32
	// +kubebuilder:validation:Minimum=0
	// Replicas of CiscoAciAim API to run
	Replicas *int32 `json:"replicas"`

	// AciConnection contains all settings related to connecting to the APIC.
	// This entire block is required for the operator to function.
	// +kubebuilder:validation:Required
	AciConnection AciConnectionSpec `json:"aciConnection"`

	// AciFabric contains fabric integration and topology settings.
	// +kubebuilder:validation:Optional
	AciFabric *AciFabricSpec `json:"aciFabric,omitempty"`

	// Whether to enable sending of gratarp on GEN1 hardware for GARP optimization.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	AciGen1HwGratArps bool `json:"AciGen1HwGratArps,omitempty"`

	// Whether to subscribe to APIC faults for monitoring.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	AciEnableFaultSubscription bool `json:"AciEnableFaultSubscription,omitempty"`

	// Enable debug logging for AIM services.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	ACIAimDebug bool `json:"ACIAimDebug,omitempty"`

	// Enable Optimized Metadata service.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	ACIOptimizedMetadata bool `json:"ACIOptimizedMetadata,omitempty"`

	// Enable scoping of names with apic_system_id.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	ACIScopeNames bool `json:"ACIScopeNames,omitempty"`

	// Enable scoping of Infra names with apic_system_id.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	ACIScopeInfra bool `json:"ACIScopeInfra,omitempty"`

	// +kubebuilder:validation:Required
	LogPersistence LogPersistenceSpec `json:"logPersistence"`

	// LivenessProbe defines the liveness probe configuration for the AIM container.
	// +kubebuilder:validation:Optional
	LivenessProbe *LivenessProbeSpec `json:"livenessProbe,omitempty"`
}

// CiscoAciAimStatus defines the observed state of CiscoAciAim
type CiscoAciAimStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CiscoAciAim is the Schema for the ciscoaciaims API
type CiscoAciAim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CiscoAciAimSpec   `json:"spec,omitempty"`
	Status CiscoAciAimStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CiscoAciAimList contains a list of CiscoAciAim
type CiscoAciAimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CiscoAciAim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CiscoAciAim{}, &CiscoAciAimList{})
}
