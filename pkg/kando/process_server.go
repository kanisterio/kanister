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

	"github.com/kanisterio/kanister/pkg/kanx"
)

func newProcessServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "start the KanX server",
		Args:  cobra.NoArgs,
		RunE:  runProcessServer,
	}
	return cmd
}

func runProcessServer(cmd *cobra.Command, args []string) error {
	address, err := processAddressFlagValue(cmd)
	if err != nil {
		return err
	}
	cmd.SilenceUsage = true

	return kanx.NewServer().Serve(cmd.Context(), address)
}
