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

// UniverseSpec defines the desired state of Universe
type UniverseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ProjectID that this unviverse sitting in. Edit universe_types.go to remove/update
	ProjectID string `json:"projectId,omitempty"`
	// TTL is the amount of time an environment exist before expired
	TTL int `json:"ttl,omitempty"`
	// Container is a nested struct for gcp container resources
	Container Container `json:"container,omitempty"`
	// Compute is a nested struct for gcp compute resources
	Compute Compute `json:"compute,omitempty"`
	// SQL is a nested struct for gcp sql resources
	SQL SQL `json:"sql,omitempty"`
	// Storage is a nested struct for gcp storage resources
	Storage Storage `json:"storage,omitempty"`
	// Storage is a struct for gcp sql instance restored from backup
	SQLRestore SQLRestore `json:"sqlRestore,omitempty"`
	// IAM is a struct for IAM resources
	IAM IAM `json:"iam,omitempty"`
	// TargetNamespaces are list of namespace to do snapshot
	TargetNamespaces []string `json:"targetNamespaces,omitempty"`
	// PrimaryCluster is the cluster to create snapshot from
	PrimaryCluster string `json:"primaryCluster,omitempty"`
}

type ResourceStatus struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// UniverseStatus defines the observed state of Universe
type UniverseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Conditions represent the latest available observations of an object's state
	// The generation observed by controller.
	// +optional
	Resources          []ResourceStatus `json:"resources,omitempty"`
	ObservedGeneration int64            `json:"observedGeneration,omitempty" protobuf:"varint,1,opt,name=observedGeneration"`
	Repository         string           `json:"repo,omitempty"`
	PR                 string           `json:"prNumber,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Repo",type=string,JSONPath=`.status.repo`
// +kubebuilder:printcolumn:name="PR",type=string,JSONPath=`.status.prNumber`
// Universe is the Schema for the universes API
type Universe struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UniverseSpec   `json:"spec,omitempty"`
	Status UniverseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// UniverseList contains a list of Universe
type UniverseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Universe `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Universe{}, &UniverseList{})
}
