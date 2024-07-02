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

package repository

import (
	"context"

	"github.com/kanisterio/errkit"
	"k8s.io/client-go/kubernetes"

	"github.com/kanisterio/kanister/pkg/kopia/command"
	kerrors "github.com/kanisterio/kanister/pkg/kopia/errors"
)

// ConnectToOrCreateKopiaRepository connects to a kopia repository if present or creates if not already present
func ConnectToOrCreateKopiaRepository(
	ctx context.Context,
	cli kubernetes.Interface,
	namespace,
	pod,
	container string,
	cmdArgs command.RepositoryCommandArgs,
) error {
	err := ConnectToKopiaRepository(
		ctx,
		cli,
		namespace,
		pod,
		container,
		cmdArgs,
	)
	switch {
	case err == nil:
		// If repository connect was successful, we're done!
		return nil
	case kerrors.IsInvalidPasswordError(err):
		// If connect failed due to invalid password, no need to attempt creation
		return err
	}

	// Create a new repository
	err = CreateKopiaRepository(
		ctx,
		cli,
		namespace,
		pod,
		container,
		cmdArgs,
	)

	if err == nil {
		// Successfully created repository, we're done!
		return nil
	}

	// Creation failed. Repository may already exist or may have been
	// created by some parallel operation. Attempt connecting again.
	connectErr := ConnectToKopiaRepository(
		ctx,
		cli,
		namespace,
		pod,
		container,
		cmdArgs,
	)

	// Connected successfully after all
	if connectErr == nil {
		return nil
	}

	err = errkit.Append(err, connectErr)
	return err
}
