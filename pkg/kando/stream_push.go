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

const (
	sourcePathFlagName = "path"
	fileNameFlagName   = "file"
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
	cmd.Flags().StringP(fileNameFlagName, "f", "", "Specify a name for the data stream (required)")
	cmd.Flags().StringP(sourcePathFlagName, "d", "", "Specify a directory path for the data stream (required)")
	_ = cmd.MarkFlagRequired(fileNameFlagName)
	_ = cmd.MarkFlagRequired(sourcePathFlagName)
	return cmd
}

func runStreamPush(cmd *cobra.Command, args []string) error {
	// TODO: Implement stream push
	_ = passwordFlag(cmd)
	_ = fileNameFlag(cmd)
	_ = sourcePathFlag(cmd)
	_ = args[0]
	return nil
}

func fileNameFlag(cmd *cobra.Command) string {
	return cmd.Flag(fileNameFlagName).Value.String()
}

func sourcePathFlag(cmd *cobra.Command) string {
	return cmd.Flag(sourcePathFlagName).Value.String()
}

// GenerateStreamPushCommand generates a bash command for
// kando stream push with given flags and arguments
func GenerateStreamPushCommand(fileName, password, path, sourceEndpoint string) []string {
	kandoCmd := []string{
		"kando",
		"stream",
		"push",
		"-p",
		password,
		"-f",
		fileName,
		"-d",
		path,
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
