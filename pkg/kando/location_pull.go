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
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	kopiaSnapshotFlagName = "kopia-snapshot"
)

func newLocationPullCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull <target>",
		Short: "Pull from s3-compliant object storage to a file or stdout",
		Args:  cobra.ExactArgs(1),
		// TODO: Example invocations
		RunE: func(c *cobra.Command, args []string) error {
			if err := validateCommandArgs(c); err != nil {
				return err
			}
			dataMover, err := dataMoverFromCMD(c)
			if err != nil {
				return err
			}
			ctx := context.Background()
			return dataMover.Pull(ctx, args[0], pathFlag(c))
		},
	}
	cmd.Flags().StringP(kopiaSnapshotFlagName, "k", "", "Pass the kopia snapshot information from the location push command (optional)")
	return cmd
}

func targetWriter(target string) (io.Writer, error) {
	if target != usePipeParam {
		return os.OpenFile(target, os.O_RDWR|os.O_CREATE, 0755)
	}
	return os.Stdout, nil
}

func locationPull(ctx context.Context, p *param.Profile, path string, target io.Writer) error {
	return location.Read(ctx, target, *p, path)
}

func kopiaSnapshotFlag(cmd *cobra.Command) string {
	return cmd.Flags().Lookup(kopiaSnapshotFlagName).Value.String()
}
