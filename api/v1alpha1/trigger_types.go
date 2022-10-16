/*
Copyright 2022.

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

// TriggerSpec defines the desired state of Trigger
type TriggerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Trigger. Edit trigger_types.go to remove/update
	Respositories []string `json:"repositories,omitempty"`
	// ProjectID that this unviverse sitting in. Edit universe_types.go to remove/update
	ProjectID           string `json:"projectId,omitempty"`
	CredentialSecretRef string `json:"credentialSecretRef,omitempty"`
	RequeueAfterMinutes int    `json:"requeueAfterMinutes,omitempty"`
	// TargetNamespaces are list of namespace to do snapshot
	TargetNamespaces []string `json:"targetNamespaces,omitempty"`
	// PrimaryCluster is main cluster to clone from
	PrimaryCluster string `json:"primaryCluster,omitempty"`
	// LabelToDeploymentMap is a map from github pr label to kubernetes deployment name
	LabelToDeploymentMap map[string]string `json:"labelToDeploymentMap,omitempty"`
}

// TriggerStatus defines the observed state of Trigger
type TriggerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,1,opt,name=observedGeneration"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Trigger is the Schema for the triggers API
type Trigger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TriggerSpec   `json:"spec,omitempty"`
	Status TriggerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TriggerList contains a list of Trigger
type TriggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Trigger `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Trigger{}, &TriggerList{})
}
