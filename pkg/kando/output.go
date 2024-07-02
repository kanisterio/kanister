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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/output"
)

func newOutputCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "output <key> <value>",
		Short: "Create phase output with given key:value",
		Args:  validateArguments,
		// TODO: Example invocations
		RunE: runOutputCommand,
	}
	return cmd
}

func validateArguments(c *cobra.Command, args []string) error {
	if len(args) != 2 {
		return errors.Errorf("Command accepts 2 arguments, received %d arguments", len(args))
	}
	return output.ValidateKey(args[0])
}

func runOutputCommand(c *cobra.Command, args []string) error {
	return output.PrintOutput(args[0], args[1])
}
