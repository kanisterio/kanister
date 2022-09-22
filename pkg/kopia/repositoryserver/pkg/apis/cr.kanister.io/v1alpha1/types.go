// Copyright 2022 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RepositoryServer manages the lifecycle of Kopia Repository Server within a Pod
type RepositoryServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RepositoryServerSpec   `json:"spec"`
	Status            RepositoryServerStatus `json:"status"`
}

// RepositoryServerSpec is the specification for the RepositoryServer
type RepositoryServerSpec struct {
	Location   Location   `json:"location,omitempty"`
	Repository Repository `json:"repository,omitempty"`
	Server     Server     `json:"server,omitempty"`
}

// Location references the backend store where a repository already exists
// and the credential necessary to connect to the backend store
type Location struct {
	LocationSecretName   string `json:"locationSecretName,omitempty"`
	CredentialSecretName string `json:"credentialSecretName,omitempty"`
}

// Repository details for the purpose of establishing a connection
type Repository struct {
	RootPath           string `json:"rootPath,omitempty"`
	PasswordSecretName string `json:"passwordSecretName,omitempty"`
}

// Server details required for starting the repository proxy server and initializing the repository client users
type Server struct {
	AdminSecretName      string               `json:"adminSecretName,omitempty"`
	TLSSecretName        string               `json:"tlsSecretName,omitempty"`
	UserAccessSecretName string               `json:"userAccessSecretName,omitempty"`
	LabelSelector        metav1.LabelSelector `json:"labelSelector,omitempty"`
}

// RepositoryServerStatus is the status for the RepositoryServer. This should only be updated by the controller.
type RepositoryServerStatus struct {
	Conditions []Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
	ServerInfo ServerInfo  `json:"serverInfo,omitempty"`
}

// Condition contains details of the current state of the RepositoryServer resource
type Condition struct {
	LastTransitionTime metav1.Time                   `json:"lastTransitionTime,omitempty"`
	LastUpdateTime     metav1.Condition              `json:"lastUpdateTime,omitempty"`
	Status             metav1.ConditionStatus        `json:"status"`
	Type               RepositoryServerConditionType `json:"type"`
}

// RepositoryServerConditionType defines all the various condition types of the RepositoryServer resource
type RepositoryServerConditionType string

const (
	// RepositoryReady indicates whether the existing repository is connected and ready to use
	RepositoryReady RepositoryServerConditionType = "RepositoryReady"

	// ServerInitialized means that the proxy server, that serves the repository, has been started
	ServerInitialized RepositoryServerConditionType = "ServerInitialized"

	// ClientsInitialized indicates that the client users have been added or updated to the repository server
	ClientsInitialized RepositoryServerConditionType = "ClientsInitialized"

	// ServerRefreshed denotes the refreshed condition of the repository server in order to register client users
	ServerRefreshed RepositoryServerConditionType = "ServerRefreshed"

	// ServerReady represents the ready state of the repository server and the pod which runs the proxy server
	ServerReady RepositoryServerConditionType = "ServerReady"
)

// ServerInfo describes all the information required by the client users to connect to the repository server
type ServerInfo struct {
	PodName           string `json:"podName,omitempty"`
	NetworkPolicyName string `json:"networkPolicyName,omitempty"`
	ServiceName       string `json:"serviceName,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RepositoryServerList is the definition of a list of RepositoryServers
type RepositoryServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RepositoryServer `json:"items"`
}
