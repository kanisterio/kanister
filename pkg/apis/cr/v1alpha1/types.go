/*
Copyright 2019 The Kanister Authors.

Copyright 2017 The Rook Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Some of the code below came from https://github.com/coreos/etcd-operator
which also has the apache 2.0 license.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	sp "k8s.io/apimachinery/pkg/util/strategicpatch"
)

const (
	// SchemeVersion is the API version of objects in this package.
	SchemeVersion = "v1alpha1"
	// ResourceGroup is the API group of resources in this package.
	ResourceGroup = "cr.kanister.io"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: ResourceGroup, Version: SchemeVersion}

// JSONMap contains PodOverride specs.
type JSONMap sp.JSONMap

var _ runtime.Object = (*ActionSet)(nil)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ActionSet describes kanister actions.
type ActionSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	// Spec defines the specification for the actionset.
	// The specification includes a list of Actions to be performed. Each Action includes details
	// about the referenced Blueprint and other objects used to perform the defined action.
	Spec *ActionSetSpec `json:"spec,omitempty"`
	// Status refers to the current status of the Kanister actions.
	Status *ActionSetStatus `json:"status,omitempty"`
}

// ObjectReference refers to a kubernetes object.
type ObjectReference struct {
	// API version of the referent.
	APIVersion string `json:"apiVersion"`
	// API Group of the referent.
	Group string `json:"group"`
	// Resource name of the referent.
	Resource string `json:"resource"`
	// Kind of the referent.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	Kind string `json:"kind"`
	// Name of the referent.
	// More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name"`
	// Namespace of the referent.
	// More info: http://kubernetes.io/docs/user-guide/namespaces
	Namespace string `json:"namespace,omitempty"`
}

// ActionSetSpec is the specification for the actionset.
type ActionSetSpec struct {
	// Actions represents a list of Actions that need to be performed by the actionset.
	Actions []ActionSpec `json:"actions,omitempty"`
}

// ActionSpec is the specification for a single Action.
type ActionSpec struct {
	// Name is the action we'll perform. For example: `backup` or `restore`.
	Name string `json:"name"`
	// Object refers to the thing we'll perform this action on.
	Object ObjectReference `json:"object"`
	// Blueprint with instructions on how to execute this action.
	Blueprint string `json:"blueprint,omitempty"`
	// Artifacts will be passed as inputs into this phase.
	Artifacts map[string]Artifact `json:"artifacts,omitempty"`
	// ConfigMaps that we'll get and pass into the blueprint.
	ConfigMaps map[string]ObjectReference `json:"configMaps,omitempty"`
	// Secrets that we'll get and pass into the blueprint.
	Secrets map[string]ObjectReference `json:"secrets,omitempty"`
	// Profile is use to specify the location where store artifacts and the
	// credentials authorized to access them.
	Profile *ObjectReference `json:"profile,omitempty"`
	// RepositoryServer is used to specify the CR reference
	// of the kopia repository server
	RepositoryServer *ObjectReference `json:"repositoryServer,omitempty"`
	// PodOverride is used to specify pod specs that will override the
	// default pod specs
	PodOverride JSONMap `json:"podOverride,omitempty"`
	// Options will be used to specify additional values
	// to be used in the Blueprint.
	Options map[string]string `json:"options,omitempty"`
	// PreferredVersion will be used to select the preferred version of Kanister functions
	// to be executed for this action
	PreferredVersion string `json:"preferredVersion"`
	// PodLabels will be used to configure the labels of the pods that are created
	// by Kanister functions run by this ActionSet
	PodLabels map[string]string `json:"podLabels"`
	// PodAnnotations will be used to configure the annotations of the pods that created
	// by Kanister functions run by this ActionSet
	PodAnnotations map[string]string `json:"podAnnotations"`
}

// ActionSetStatus is the status for the actionset. This should only be updated by the controller.
type ActionSetStatus struct {
	// State represents the current state of the actionset.
	// There are four possible values: "Pending", "Running", "Failed", and "Complete".
	State State `json:"state"`
	// Actions list represents the latest available observations of the current state of all the actions.
	Actions []ActionStatus `json:"actions,omitempty"`
	// Error contains the detailed error message of an actionset failure.
	Error Error `json:"error,omitempty"`
	// Progress provides information on the progress of a running actionset.
	// This includes the percentage of completion of an actionset and the phase that is
	// currently being executed.
	Progress ActionProgress `json:"progress,omitempty"`
}

// ActionStatus is updated as we execute phases.
type ActionStatus struct {
	// Name is the action we'll perform. For example: `backup` or `restore`.
	Name string `json:"name"`
	// Object refers to the thing we'll perform this action on.
	Object ObjectReference `json:"object"`
	// Blueprint with instructions on how to execute this action.
	Blueprint string `json:"blueprint"`
	// Phases are sub-actions an are executed sequentially.
	Phases []Phase `json:"phases,omitempty"`
	// Artifacts created by this phase.
	Artifacts map[string]Artifact `json:"artifacts,omitempty"`
	// DeferPhase is the phase that is executed at the end of an action
	// irrespective of the status of other phases in the action
	DeferPhase Phase `json:"deferPhase,omitempty"`
}

// ActionProgress provides information on the combined progress
// of all the phases in the action.
type ActionProgress struct {
	// RunningPhase represents which phase of the action is being run
	RunningPhase string `json:"runningPhase,omitempty"`
	// PercentCompleted is computed by assessing the number of completed phases
	// against the total number of phases.
	PercentCompleted string `json:"percentCompleted,omitempty"`
	// SizeDownloadedB represents the size of data downloaded in Bytes at a given time during action execution.
	// This field will be empty for actions which do not involve data movement.
	SizeDownloadedB int64 `json:"sizeDownloadedB,omitempty"`
	// SizeUploadedB represents the size of data uploaded in Bytes at a given time during action execution.
	// This field will be empty for actions which do not involve data movement.
	SizeUploadedB int64 `json:"sizeUploadedB,omitempty"`
	// EstimatedDownloadSizeB represents the total estimated size of data in Bytes
	// that will be downloaded during the action execution.
	// This field will be empty for actions which do not involve data movement.
	EstimatedDownloadSizeB int64 `json:"estimatedDownloadSizeB,omitempty"`
	// EstimatedUploadSizeB represents the total estimated size of data in Bytes
	// that will be uploaded during the phase execution.
	// This field will be empty for phases which do not involve data movement.
	EstimatedUploadSizeB int64 `json:"estimatedUploadSizeB,omitempty"`
	// LastTransitionTime represents the last date time when the progress status
	// was received.
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`
}

// State is the current state of a phase of execution.
type State string

const (
	// StatePending mean this action or phase has yet to be executed.
	StatePending State = "pending"
	// StateRunning means this action or phase is currently executing.
	StateRunning State = "running"
	// StateFailed means this action or phase was unsuccessful.
	StateFailed State = "failed"
	// StateComplete means this action or phase finished successfully.
	StateComplete State = "complete"
)

// Error represents an error that occurred when executing an actionset.
type Error struct {
	// Message is the actual error message that is displayed in case of errors.
	Message string `json:"message"`
}

// Phase is subcomponent of an action.
type Phase struct {
	// Name represents the name of the Blueprint phase.
	Name string `json:"name"`
	// State represents the current state of execution of the Blueprint phase.
	State State `json:"state"`
	// Output is the map of output artifacts produced by the Blueprint phase.
	Output map[string]interface{} `json:"output,omitempty"`
	// Progress represents the phase execution progress.
	Progress PhaseProgress `json:"progress,omitempty"`
}

// PhaseProgress represents the execution state of the phase.
type PhaseProgress struct {
	// ProgressPercent represents the execution progress in percentage.
	ProgressPercent string `json:"progressPercent,omitempty"`
	// SizeDownloadedB represents the size of data downloaded in Bytes at a given time during phase execution.
	// This field will be empty for phases which do not involve data movement.
	SizeDownloadedB int64 `json:"sizeDownloadedB,omitempty"`
	// SizeUploadedB represents the size of data uploaded in Bytes at a given time during phase execution.
	// This field will be empty for phases which do not involve data movement.
	SizeUploadedB int64 `json:"sizeUploadedB,omitempty"`
	// EstimatedDownloadSizeB represents the total estimated size of data in Bytes
	// that will be downloaded during the phase execution.
	// This field will be empty for phases which do not involve data movement.
	EstimatedDownloadSizeB int64 `json:"estimatedDownloadSizeB,omitempty"`
	// EstimatedUploadSizeB represents the total estimated size of data in Bytes
	// that will be uploaded during the phase execution.
	// This field will be empty for phases which do not involve data movement.
	EstimatedUploadSizeB int64 `json:"estimatedUploadSizeB,omitempty"`
	// EstimatedTimeSeconds is the estimated time required in seconds to transfer the
	// remaining data estimated with EstimatedUploadSizeB/EstimatedDownloadSizeB.
	// This field will be empty for phases which do not involve data movement.
	EstimatedTimeSeconds int64 `json:"estinatedTimeSeconds,omitempty"`
	// LastTransitionTime represents the last date time when the progress status
	// was received.
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`
}

// k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Artifact tracks objects produced by an action.
type Artifact struct {
	// KeyValue represents key-value pair artifacts produced by the action.
	KeyValue map[string]string `json:"keyValue,omitempty"`
	// KopiaSnapshot captures the kopia snapshot information
	// produced as a JSON string by kando command in phases of an action.
	KopiaSnapshot string `json:"kopiaSnapshot,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ActionSetList is the definition of a list of actionsets.
type ActionSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	// Items is the list of actionsets.
	Items []*ActionSet `json:"items"`
}

var _ runtime.Object = (*Blueprint)(nil)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Blueprint describes kanister actions.
type Blueprint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	// Actions is the list of actions constructing the Blueprint.
	Actions map[string]*BlueprintAction `json:"actions,omitempty"`
}

// BlueprintAction describes the set of phases that constitute an action.
type BlueprintAction struct {
	// Name contains the name of the action.
	Name string `json:"name"`
	// Kind contains the resource on which this action has to be performed.
	Kind string `json:"kind"`
	// ConfigMapNames is used to specify the config map names that can be used later in the action phases.
	ConfigMapNames []string `json:"configMapNames,omitempty"`
	// List of Kubernetes secret names used in action phases.
	SecretNames []string `json:"secretNames,omitempty"`
	// InputArtifactNames is the list of Artifact names that were set from previous action and can be consumed in the current action.
	InputArtifactNames []string `json:"inputArtifactNames,omitempty"`
	// OutputArtifacts is the map of rendered artifacts produced by the BlueprintAction.
	OutputArtifacts map[string]Artifact `json:"outputArtifacts,omitempty"`
	// Phases is the list of BlueprintPhases which are invoked in order when executing this action.
	Phases []BlueprintPhase `json:"phases,omitempty"`
	// DeferPhase is invoked after the execution of Phases that are defined for an action.
	// A DeferPhase is executed regardless of the statuses of the other phases of the action.
	// A DeferPhase can be used for cleanup operations at the end of an action.
	DeferPhase *BlueprintPhase `json:"deferPhase,omitempty"`
}

// BlueprintPhase is a an individual unit of execution.
type BlueprintPhase struct {
	// Func is the name of a registered Kanister function.
	Func string `json:"func"`
	// Name contains name of the phase.
	Name string `json:"name"`
	// ObjectRefs represents a map of references to the Kubernetes objects that
	// can later be used in the `Args` of the function.
	ObjectRefs map[string]ObjectReference `json:"objects,omitempty"`
	// Args represents a map of named arguments that the controller will pass to the Kanister function.
	Args map[string]interface{} `json:"args"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BlueprintList is the definition of a list of Blueprints
type BlueprintList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	// Items is the list of Blueprints.
	Items []*Blueprint `json:"items"`
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Profile captures information about a storage location for backup artifacts and
// corresponding credentials, that will be made available to a Blueprint phase.
type Profile struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	// Location provides the information about the object storage that is going to be used by Kanister to upload the backup objects.
	Location Location `json:"location"`
	// Credential represents the credentials associated with the Location.
	Credential Credential `json:"credential"`
	// SkipSSLVerify is a boolean that specifies whether skipping SSL verification
	// is allowed when operating with the Location.
	// If omitted from the CR definition, it defaults to false
	SkipSSLVerify bool `json:"skipSSLVerify"`
}

type LocationType string

const (
	LocationTypeGCS         LocationType = "gcs"
	LocationTypeS3Compliant LocationType = "s3Compliant"
	LocationTypeAzure       LocationType = "azure"
	LocationTypeKopia       LocationType = "kopia"
)

type Location struct {
	// Type specifies the kind of object storage that would be used to upload the
	// backup objects. Currently supported values are: "GCS", "S3Compliant",
	// and "Azure".
	Type LocationType `json:"type"`
	// Bucket represents the bucket on the object storage where the backup is uploaded.
	Bucket string `json:"bucket"`
	// Endpoint specifies the endpoint where the object storage is accessible at.
	Endpoint string `json:"endpoint"`
	// Prefix is the string that would be prepended to the object path in the
	// bucket where the backup objects are uploaded.
	Prefix string `json:"prefix"`
	// Region represents the region of the bucket specified above.
	Region string `json:"region"`
}

type CredentialType string

const (
	CredentialTypeKeyPair CredentialType = "keyPair"
	CredentialTypeSecret  CredentialType = "secret"
	CredentialTypeKopia   CredentialType = "kopia"
)

type Credential struct {
	// Type represents the information about how the credentials are provided for the respective object storage.
	Type CredentialType `json:"type"`
	// KeyPair represents the key-value map used for the Credential of Type KeyPair.
	KeyPair *KeyPair `json:"keyPair,omitempty"`
	// Secret represents the Kubernetes Secret Object used for the Credential of Type Secret.
	Secret *ObjectReference `json:"secret,omitempty"`
	// KopiaServerSecret represents the secret being used by Credential of Type Kopia.
	KopiaServerSecret *KopiaServerSecret `json:"kopiaServerSecret,omitempty"`
}

type KeyPair struct {
	// IDField specifies the corresponding key in the secret where the AWS Key ID value is stored.
	IDField string `json:"idField"`
	// SecretField specifies the corresponding key in the secret where the AWS Secret Key value is stored.
	SecretField string `json:"secretField"`
	// Secret represents a Kubernetes Secret object storing the KeyPair credentials.
	Secret ObjectReference `json:"secret"`
}

// KopiaServerSecret contains credentials to connect to Kopia server
type KopiaServerSecret struct {
	// Username represents the username used to connect to the Kopia Server.
	Username string `json:"username,omitempty"`
	// Hostname represents the hostname used to connect to the Kopia Server.
	Hostname string `json:"hostname,omitempty"`
	// UserPassphrase is the user password used to connect to the Kopia Server.
	UserPassphrase *KopiaServerSecretRef `json:"userPassphrase,omitempty"`
	// TLSCert is the certificate used to connect to the Kopia Server.
	TLSCert *KopiaServerSecretRef `json:"tlsCert,omitempty"`
	// ConnectOptions represents a map of options which can be used to connect to the Kopia Server.
	ConnectOptions map[string]int `json:"connectOptions,omitempty"`
}

// KopiaServerSecretRef refers to K8s secrets containing Kopia creds
type KopiaServerSecretRef struct {
	// Key represents the corresponding key in the secret where the required
	// credential or certificate value is stored.
	Key string `json:"key"`
	// Secret is the K8s secret object where the creds related to the Kopia Server are stored.
	Secret *ObjectReference `json:"secret"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProfileList is the definition of a list of Profiles
type ProfileList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	// Items represents a list of Profiles.
	Items []*Profile `json:"items"`
}
