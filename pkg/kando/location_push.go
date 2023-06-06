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

	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/location"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	outputNameFlagName    = "output-name"
	defaultKandoOutputKey = "kandoOutput"
)

func newLocationPushCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push <source>",
		Short: "Push a source file or stdin stream to s3-compliant object storage",
		Args:  cobra.ExactArgs(1),
		// TODO: Example invocations
		RunE: func(c *cobra.Command, args []string) error {
			if err := validateCommandArgs(c); err != nil {
				return err
			}
			dataMover, err := dataMoverFromCMD(c, outputNameFlagName)
			if err != nil {
				return err
			}
			ctx := context.Background()
			return dataMover.Push(ctx, args[0], pathFlag(c))
		},
	}
	cmd.Flags().StringP(outputNameFlagName, "o", defaultKandoOutputKey, "Specify a name to be used for the output produced by kando. Set to `kandoOutput` by default")

	return cmd
}

const usePipeParam = `-`

func locationPush(ctx context.Context, p *param.Profile, path string, source io.Reader) error {
	return location.Write(ctx, source, *p, path)
}

func outputNameFlag(cmd *cobra.Command) string {
	return cmd.Flags().Lookup(outputNameFlagName).Value.String()
}
