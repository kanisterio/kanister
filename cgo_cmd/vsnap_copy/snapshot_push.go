// Copyright 2020 The Kanister Authors.
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

package main

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/astrolabe/pkg/astrolabe"

	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
)

func newSnapshotPushCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push <source>",
		Short: "Push a source file or stdin stream to s3-compliant object storage",
		Args:  cobra.ExactArgs(1),
		// TODO: Example invocations
		RunE: func(c *cobra.Command, args []string) error {
			return runSnapshotPush(c, args)
		},
	}
	return cmd
}

func runSnapshotPush(cmd *cobra.Command, args []string) error {
	snapshotID := args[0]
	profile, err := unmarshalProfileFlag(cmd)
	if err != nil {
		return err
	}
	path := pathFlag(cmd)
	config, err := unmarshalVSphereCredentials(cmd)
	if err != nil {
		return err
	}
	ctx := context.Background()
	return copySnapshotToObjectStore(ctx, config, profile, snapshotID, path)
}

func copySnapshotToObjectStore(ctx context.Context, config *VSphereCreds, profile *param.Profile, snapshot string, path string) error {
	snapManager, err := NewSnapshotManager(config)
	if err != nil {
		return err
	}

	// expecting a snapshot id of the form type:volumeID:snapshotID
	peID, err := astrolabe.NewProtectedEntityIDFromString(snapshot)
	if err != nil {
		return err
	}
	pe, err := snapManager.ivdPETM.GetProtectedEntity(ctx, peID)
	if err != nil {
		return err
	}

	reader, err := pe.GetDataReader(ctx)
	if err != nil {
		return err
	}

	return location.Write(ctx, reader, *profile, path)
}
