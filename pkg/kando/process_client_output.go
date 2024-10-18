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
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/kanx"
)

func newProcessClientOutputCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "output",
		Short: "output",
		RunE:  runProcessClientOutput,
	}
	return cmd
}

func runProcessClientOutput(cmd *cobra.Command, args []string) error {
	cmd.SetContext(context.Background())
	pid, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}
	addr, err := processAddressFlagValue(cmd)
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true
	return kanx.Stdout(cmd.Context(), addr, int64(pid), os.Stdout)
}
