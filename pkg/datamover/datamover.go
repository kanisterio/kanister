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

package datamover

type DataMover interface {
	// Pull is used to restore the data from object storage
	// using the preferred data-mover
	Pull(sourcePath, destinationPath string) error
	// Push is used to backup the data to object storage
	// using the preferred data-mover
	Push(sourcePath, destinationPath string) error
	// Delete is used to delete the data from object storage
	// using the preferred data-mover
	Delete(destinationPath string) error
}
