package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of ApplicationHealth
	// +required
	Spec ApplicationHealthSpec `json:"spec,omitempty"`

	// status defines the observed state of ApplicationHealth
	// +optional
	Status ApplicationHealthStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ApplicationHealthList contains a list of ApplicationHealth
type ApplicationHealthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []ApplicationHealth `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &ApplicationHealth{}, &ApplicationHealthList{})
		return nil
	})
}
