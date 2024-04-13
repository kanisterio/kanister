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
	"net"

	"github.com/spf13/cobra"
)

const (
	processAddressFlagName = "address"
)

func newProcessCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "process <command>",
		Short: "Manage kando processes",
	}
	cmd.AddCommand(newProcessServerCommand())
	cmd.AddCommand(newProcessClientCommand())
	cmd.PersistentFlags().StringP(processAddressFlagName, "a", "/tmp/kanister.sock", "The path of a unix socket of the process server")
	return cmd
}

func processAddressFlagValue(cmd *cobra.Command) (string, error) {
	a := cmd.Flag(processAddressFlagName).Value.String()
	_, err := net.ResolveUnixAddr("unix", a)
	return a, err
}
