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

	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
)

func newLocationDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete artifacts from s3-compliant object storage",
		// TODO: Example invocations
		RunE: func(c *cobra.Command, args []string) error {
			if err := validateCommandArgs(c); err != nil {
				return err
			}
			dataMover, err := dataMoverFromCMD(c, kopiaSnapshotFlagName)
			if err != nil {
				return err
			}
			return dataMover.Delete(context.Background(), pathFlag(c))
		},
	}
	cmd.Flags().StringP(kopiaSnapshotFlagName, "k", "", "Pass the kopia snapshot information from the location push command (optional)")
	return cmd
}

func locationDelete(ctx context.Context, p *param.Profile, path string) error {
	return location.Delete(ctx, *p, path)
}
