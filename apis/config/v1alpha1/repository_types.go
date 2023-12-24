/*
Copyright 2024 Nokia.

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

type RepositoryType string

const (
	RepositoryTypeGit RepositoryType = "git"
	RepositoryTypeOCI RepositoryType = "oci"
)

// RepositorySpec defines the desired state of Repository
type RepositorySpec struct {
	// Type of the repository (i.e. git, OCI)
	// +kubebuilder:validation:Enum=git;oci
	// +kubebuilder:default:="git"
	Type RepositoryType `json:"type,omitempty" yaml:"type,omitempty"`
	// Git repository details. Required if `type` is `git`. Ignored if `type` is not `git`.
	Git *GitRepository `json:"git,omitempty" yaml:"git,omitempty"`
	// OCI repository details. Required if `type` is `oci`. Ignored if `type` is not `oci`.
	Oci *OciRepository `json:"oci,omitempty" yaml:"oci,omitempty"`
	// The repository is a deployment repository;
	// When set to true this is considered a WET package; when false this is a DRY package
	Deployment bool `json:"deployment,omitempty" yaml:"deployment,omitempty"`
}

// GitRepository describes a Git repository.
// TODO: authentication methods
type GitRepository struct {
	// URL specifies the base URL for a given repository for example:
	//   `https://github.com/GoogleCloudPlatform/blueprints.git`
	URL string `json:"url" yaml:"url"`
	// Name of the ref where we want to get the files from; can be a tag or a branch; if unspecified it points to main
	Ref string `json:"ref,omitempty"`
	// Directory within the Git repository where the files are stored. If unspecified, defaults to root directory.
	Directory string `json:"directory,omitempty"`
	// Credentials defines the name of the secret that holds the credentials to connect to a repository
	Credentials string `json:"credentials,omitempty" yaml:"credentials,omitempty"`
}

// OciRepository describes a repository compatible with the Open Container Registry standard.
// TODO: allow sub-selection of the registry, i.e. filter by tags, ...?
// TODO: authentication types?
type OciRepository struct {
	// Registry is the address of the OCI registry
	Registry string `json:"registry"`
	// Credentials defines the name of the secret that holds the credentials to connect to the OCI registry
	Credentials string `json:"credentials,omitempty" yaml:"credentials,omitempty"`
}

// RepositoryStatus defines the observed state of Repository
type RepositoryStatus struct {
	// ConditionedStatus provides the status of the Repository using conditions
	ConditionedStatus `json:",inline" yaml:",inline"`
}

// +kubebuilder:object:root=true
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
//+kubebuilder:printcolumn:name="DEPLOYMENT",type=boolean,JSONPath=`.spec.deployment`
//+kubebuilder:printcolumn:name="TYPE",type=string,JSONPath=`.spec.type`
//+kubebuilder:printcolumn:name="ADDRESS",type=string,JSONPath=`.spec['git','oci']['repo','registry']`

// +kubebuilder:resource:categories={kform,pm}
// Repository is the Repository for the Repository API
// +k8s:openapi-gen=true
type Repository struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Spec   RepositorySpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status RepositoryStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

// +kubebuilder:object:root=true
// RepositoryList contains a list of Repositorys
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type RepositoryList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items           []Repository `json:"items" yaml:"items"`
}
