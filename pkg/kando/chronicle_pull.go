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
	"encoding/json"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/chronicle"
	"github.com/kanisterio/kanister/pkg/param"
)

func newChroniclePullCommand() *cobra.Command {
	params := locationParams{}
	cmd := &cobra.Command{
		Use:   "pull <command>",
		Short: "Pull the data referenced by a chronicle manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runChroniclePull(c, params, args[0])
		},
	}
	cmd.PersistentFlags().StringVarP(&params.suffix, pathFlagName, "s", "", "Specify a path suffix (optional)")
	cmd.PersistentFlags().StringVarP(&params.profile, profileFlagName, "p", "", "Pass a Profile as a JSON string (required)")
	_ = cmd.MarkPersistentFlagRequired(profileFlagName)
	return cmd
}

type locationParams struct {
	suffix  string
	profile string
}

func unmarshalProfile(prof string) (*param.Profile, error) {
	p := &param.Profile{}
	err := json.Unmarshal([]byte(prof), p)
	return p, errors.Wrap(err, "failed to unmarshal profile")
}

//nolint:unparam
func runChroniclePull(cmd *cobra.Command, p locationParams, arg string) error {
	target, err := targetWriter(arg)
	if err != nil {
		return err
	}
	prof, err := unmarshalProfile(p.profile)
	if err != nil {
		return err
	}
	ctx := context.Background()
	return chronicle.Pull(ctx, target, *prof, p.suffix)
}

const usePipeParam = `-`

func targetWriter(target string) (io.Writer, error) {
	if target != usePipeParam {
		return os.OpenFile(target, os.O_RDWR|os.O_CREATE, 0755)
	}
	return os.Stdout, nil
}
