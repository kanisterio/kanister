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
	"time"

	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/chronicle"
)

const (
	profilePathFlagName  = "profile-path"
	artifactPathFlagName = "artifact-path"
	frequencyFlagName    = "frequency"
	envDirFlagName       = "env-dir"
)

func newChroniclePushCommand() *cobra.Command {
	params := chronicle.PushParams{}
	cmd := &cobra.Command{
		Use:   "push <command>",
		Short: "Periodically push the output of a command to object storage",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if err := params.Validate(); err != nil {
				return err
			}
			params.Command = args
			return chronicle.Push(params)
		},
	}
	cmd.PersistentFlags().StringVarP(&params.ProfilePath, profilePathFlagName, "p", "", "Path to a Profile as a JSON string (required)")
	_ = cmd.MarkPersistentFlagRequired(profilePathFlagName)
	cmd.PersistentFlags().StringVarP(&params.ArtifactFile, artifactPathFlagName, "s", "", "Specify a file that contains an object store suffix")
	_ = cmd.MarkPersistentFlagRequired(artifactPathFlagName)
	cmd.PersistentFlags().StringVarP(&params.EnvDir, envDirFlagName, "e", "", "Get environment variables from a envdir style directory(optional)")
	cmd.PersistentFlags().DurationVarP(&params.Frequency, frequencyFlagName, "f", time.Minute, "The Frequency to push to object storage ")
	return cmd
}
