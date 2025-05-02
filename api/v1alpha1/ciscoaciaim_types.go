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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
const (
    // ContainerImage - default fall-back container image for CiscoAciAim if associated env var not provided
    ContainerImage = "10.30.120.60:8787/osp18/openstack-ciscoaci-aim:latest"
)


// CiscoAciAimSpec defines the desired state of AciAim
type CiscoAciAimSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
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
    ACIHost string `json:"aciHost"`
    ACIUser string `json:"aciUser"`
    ACIPassword string `json:"aciPassword"`
    RabbitMQCluster string `json:"rabbitmqCluster"`
    MariaDBInstance string `json:"mariadbInstance"`
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
