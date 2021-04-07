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

	"github.com/kanisterio/kanister/pkg/kopia"
	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	backupIDFlagName = "backupID"
)

func newLocationPullCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull <target>",
		Short: "Pull from s3-compliant object storage to a file or stdout",
		Args:  cobra.ExactArgs(1),
		// TODO: Example invocations
		RunE: func(c *cobra.Command, args []string) error {
			return runLocationPull(c, args)
		},
	}
	cmd.Flags().StringP(backupIDFlagName, "b", "", "Pass the backup ID from the location push command (optional)")
	return cmd
}

// nolint:unused,deadcode
func backupIDFlag(cmd *cobra.Command) string {
	return cmd.Flag(backupIDFlagName).Value.String()
}

func runLocationPull(cmd *cobra.Command, args []string) error {
	target, err := targetWriter(args[0])
	if err != nil {
		return err
	}
	p, err := unmarshalProfileFlag(cmd)
	if err != nil {
		return err
	}
	s := pathFlag(cmd)
	ctx := context.Background()
	return locationPull(ctx, p, s, target)
}

func targetWriter(target string) (io.Writer, error) {
	if target != usePipeParam {
		return os.Open(target)
	}
	return os.Stdout, nil
}

func locationPull(ctx context.Context, p *param.Profile, path string, target io.Writer) error {
	return location.Read(ctx, target, *p, path)
}

// kopiaLocationPull pulls the data from a kopia snapshot into the given target
// nolint:unused,deadcode
func kopiaLocationPull(ctx context.Context, backupID, path string, target io.Writer) error {
	return kopia.Read(ctx, backupID, path, target)
}
