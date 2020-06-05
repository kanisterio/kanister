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
	"github.com/spf13/cobra"
)

const (
	passwordFlagName = "password"
)

func newStreamCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stream <command>",
		Short: "Manage data streams in object storage",
	}
	cmd.AddCommand(newStreamPushCommand())
	cmd.PersistentFlags().StringP(passwordFlagName, "p", "", "Specify the password for object storage repository (required)")
	_ = cmd.MarkPersistentFlagRequired(passwordFlagName)
	return cmd
}

func passwordFlag(cmd *cobra.Command) string {
	return cmd.Flag(passwordFlagName).Value.String()
}
