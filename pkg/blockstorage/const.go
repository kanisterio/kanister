// Copyright 2019 The Kanister Authors.
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

package blockstorage

// Type is the type of storage supported
type Type string

const (
	// TypeAD captures enum value "AD"
	TypeAD Type = "AD"
	// TypeEBS captures enum value "EBS"
	TypeEBS Type = "EBS"
	// TypeGPD captures enum value "GPD"
	TypeGPD Type = "GPD"
	// TypeCinder captures enum value "Cinder"
	TypeCinder Type = "Cinder"
	// TypeGeneric captures enum value "Generic"
	TypeGeneric Type = "Generic"
	// TypeCeph captures enum value "Ceph"
	TypeCeph Type = "Ceph"
	// TypeEFS captures enum value "EFS"
	TypeEFS Type = "EFS"
	// TypeFCD capture enum value for "VMWare FCD"
	TypeFCD Type = "FCD"
)
