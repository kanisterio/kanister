// Copyright 2023 The Kanister Authors.
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

// Package consts declares all the constants.
package consts

const (
	ActionsetNameKey = "ActionSet"
	PodNameKey       = "Pod"
	ContainerNameKey = "Container"
	PhaseNameKey     = "Phase"
	LogKindKey       = "LogKind"
	LogKindDatapath  = "datapath"

	GoogleCloudCredsFilePath = "/tmp/creds.txt"
	LabelKeyCreatedBy        = "createdBy"
	LabelValueKanister       = "kanister"
	LabelPrefix              = "kanister.io/"
	LabelSuffixJobID         = "JobID"
)

// These names are used to query ActionSet API objects.
const (
	ActionSetResourceName       = "actionset"
	ActionSetResourceNamePlural = "actionsets"
	BlueprintResourceName       = "blueprint"
	BlueprintResourceNamePlural = "blueprints"
	ProfileResourceName         = "profile"
	ProfileResourceNamePlural   = "profiles"
)

const (
	PVProvisionedByAnnotation = "pv.kubernetes.io/provisioned-by"

	AWSEBSProvisionerInTree = "kubernetes.io/aws-ebs"
	GCEPDProvisionerInTree  = "kubernetes.io/gce-pd"
)

// These consts are used to query Repository server API objects
const (
	RepositoryServerResourceName       = "repositoryserver"
	RepositoryServerResourceNamePlural = "repositoryservers"
)

const (
	LatestKanisterToolsImage = "ghcr.io/kanisterio/kanister-tools:v9.99.9-dev"
	KanisterToolsImage       = "ghcr.io/kanisterio/kanister-tools:0.110.0"
)

// KanisterToolsImageEnvName is used to set up a custom kanister-tools image
const KanisterToolsImageEnvName = "KANISTER_TOOLS"
