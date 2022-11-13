/*
Copyright 2021.

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
	"github.com/argoproj/gitops-engine/pkg/health"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GitHubDeploymentSpec defines the desired state of GitHubDeployment
type GitHubDeploymentSpec struct {
	// GitHub Deployment URL in the form of https://api.github.com/repos/OWNER/REPO/deployments/ID
	DeploymentURL string `json:"url,omitempty"`
}

// GitHubDeploymentStatus defines the observed state of GitHubDeployment
type GitHubDeploymentStatus struct {
	LastHealthEvent GitHubDeploymentStatusHealthEvent `json:"lastHealthEvent,omitempty"`
}

type GitHubDeploymentStatusHealthEvent struct {
	UpdatedAt metav1.Time `json:"updatedAt,omitempty"`

	Health health.HealthStatusCode `json:"health,omitempty"`

	DeploymentURL string `json:"deploymentURL,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// GitHubDeployment is the Schema for the githubdeployments API
type GitHubDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GitHubDeploymentSpec   `json:"spec,omitempty"`
	Status GitHubDeploymentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GitHubDeploymentList contains a list of GitHubDeployment
type GitHubDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GitHubDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GitHubDeployment{}, &GitHubDeploymentList{})
}
