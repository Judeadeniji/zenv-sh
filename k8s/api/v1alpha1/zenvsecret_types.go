/*
Copyright 2026.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ZEnvSecretSpec defines the desired state of ZEnvSecret
type ZEnvSecretSpec struct {
	// ProjectID is the zEnv project ID
	// +kubebuilder:validation:Required
	ProjectID string `json:"projectId"`

	// Environment is the environment name to fetch (e.g., development, production)
	// +kubebuilder:validation:Required
	Environment string `json:"environment"`

	// TargetSecret is the name of the v1.Secret to create or update in the same namespace
	// +kubebuilder:validation:Required
	TargetSecret string `json:"targetSecret"`

	// CredentialsRef points to a namespace-local Secret containing 'token' and 'projectKey'
	// +optional
	CredentialsRef *corev1.LocalObjectReference `json:"credentialsRef,omitempty"`
}

// ZEnvSecretStatus defines the observed state of ZEnvSecret.
type ZEnvSecretStatus struct {
	// LastSyncedAt is the timestamp of the last successful sync
	// +optional
	LastSyncedAt *metav1.Time `json:"lastSyncedAt,omitempty"`

	// Conditions represent the current state of the ZEnvSecret resource.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ZEnvSecret is the Schema for the zenvsecrets API
type ZEnvSecret struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of ZEnvSecret
	// +required
	Spec ZEnvSecretSpec `json:"spec"`

	// status defines the observed state of ZEnvSecret
	// +optional
	Status ZEnvSecretStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ZEnvSecretList contains a list of ZEnvSecret
type ZEnvSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []ZEnvSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ZEnvSecret{}, &ZEnvSecretList{})
}
