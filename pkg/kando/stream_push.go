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

package kando

import (
	"strings"

	"github.com/spf13/cobra"
)

func newStreamPushCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push <source>",
		Short: "Push the output of a stream source to object storage",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runStreamPush(c, args)
		},
	}
	return cmd
}

func runStreamPush(cmd *cobra.Command, args []string) error {
	// TODO: Implement stream push
	_ = passwordFlag(cmd)
	_ = args[0]
	return nil
}

// GenerateStreamPushCommand generates a bash command for
// kando stream push with given password and source
func GenerateStreamPushCommand(password, sourceEndpoint string) []string {
	kandoCmd := []string{
		"kando",
		"stream",
		"push",
		"-p",
		password,
		sourceEndpoint,
	}
	return []string{
		"bash",
		"-o",
		"errexit",
		"-o",
		"pipefail",
		"-c",
		strings.Join(kandoCmd, " "),
	}
}
