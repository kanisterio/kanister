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

package kando

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
)

func newLocationDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete artifacts from s3-compliant object storage",
		// TODO: Example invocations
		RunE: func(c *cobra.Command, args []string) error {
			return runLocationDelete(c)
		},
	}
	cmd.Flags().StringP(kopiaSnapshotFlagName, "k", "", "Pass the kopia snapshot information from the location push command (optional)")
	return cmd
}

func runLocationDelete(cmd *cobra.Command) error {
	p, err := unmarshalProfileFlag(cmd)
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true
	s := pathFlag(cmd)
	ctx := context.Background()
	if p.Location.Type == crv1alpha1.LocationTypeKopia {
		snapJSON := kopiaSnapshotFlag(cmd)
		if snapJSON == "" {
			return errors.New("kopia snapshot information is required to delete data using kopia")
		}
		kopiaSnap, err := kopia.UnmarshalKopiaSnapshot(snapJSON)
		if err != nil {
			return err
		}
		if err = connectToKopiaServer(ctx, p); err != nil {
			return err
		}
		return kopiaLocationDelete(ctx, kopiaSnap.BackupID, s)
	}
	return locationDelete(ctx, p, s)
}

// kopiaLocationDelete deletes the kopia snapshot with given backupID
func kopiaLocationDelete(ctx context.Context, backupID, path string) error {
	return kopia.DeleteSnapshot(ctx, backupID, path)
}

func locationDelete(ctx context.Context, p *param.Profile, path string) error {
	return location.Delete(ctx, *p, path)
}
