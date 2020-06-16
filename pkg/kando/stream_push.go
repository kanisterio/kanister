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
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/stream"
)

const (
	streamPushDirPathFlagName  = "dirPath"
	streamPushFilePathFlagName = "filePath"
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
	cmd.Flags().StringP(streamPushDirPathFlagName, "d", "", "Specify a root directory path for the data stream (required)")
	cmd.Flags().StringP(streamPushFilePathFlagName, "f", "", "Specify a file name or path for the data stream (required)")
	_ = cmd.MarkFlagRequired(streamPushDirPathFlagName)
	_ = cmd.MarkFlagRequired(streamPushFilePathFlagName)
	return cmd
}

func runStreamPush(cmd *cobra.Command, args []string) error {
	dirPath := streamPushDirPathFlag(cmd)
	filePath := streamPushFilePathFlag(cmd)
	pwd := streamPasswordFlag(cmd)
	sourceEndpoint := args[0]
	return stream.Push(context.Background(), dirPath, filePath, pwd, sourceEndpoint)
}

func streamPushDirPathFlag(cmd *cobra.Command) string {
	return cmd.Flag(streamPushDirPathFlagName).Value.String()
}

func streamPushFilePathFlag(cmd *cobra.Command) string {
	return cmd.Flag(streamPushFilePathFlagName).Value.String()
}

// GenerateStreamPushCommand generates a bash command for
// kando stream push with given flags and arguments
func GenerateStreamPushCommand(dirPath, filePath, password, sourceEndpoint string) []string {
	kandoCmd := []string{
		"kando",
		"stream",
		"push",
		"-p",
		password,
		"-d",
		dirPath,
		"-f",
		filePath,
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
