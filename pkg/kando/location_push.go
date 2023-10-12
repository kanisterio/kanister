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
	"github.com/spf13/cobra"
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
			dataMover, err := dataMoverForOutputNameFlag(c)
			if err != nil {
				return err
			}
			return dataMover.Push(c.Context(), args[0], pathFlag(c))
		},
	}
	cmd.Flags().StringP(outputNameFlagName, "o", defaultKandoOutputKey, "Specify a name to be used for the output produced by kando. Set to `kandoOutput` by default")

	return cmd
}
