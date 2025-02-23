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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ApplicationHealthSpec defines the desired state of ApplicationHealth
type ApplicationHealthSpec struct {
}

// ApplicationHealthStatus defines the observed state of ApplicationHealth
type ApplicationHealthStatus struct {
	// Last revision when the application is healthy.
	// +optional
	LastHealthyRevision string `json:"lastHealthyRevision,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ApplicationHealth is the Schema for the applicationhealths API
type ApplicationHealth struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationHealthSpec   `json:"spec,omitempty"`
	Status ApplicationHealthStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationHealthList contains a list of ApplicationHealth
type ApplicationHealthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApplicationHealth `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ApplicationHealth{}, &ApplicationHealthList{})
}
