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
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/kanisterio/kanister/pkg/kanx"
)

func newProcessClientListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list PID and disposition of all managed processes",
		Args:  cobra.NoArgs,
		RunE:  runProcessClientList,
	}
	return cmd
}

func runProcessClientList(cmd *cobra.Command, _args []string) error {
	return runProcessClientListWithOutput(cmd.OutOrStdout(), cmd)
}

func runProcessClientListWithOutput(out io.Writer, cmd *cobra.Command) error {
	addr, err := processAddressFlagValue(cmd)
	if err != nil {
		return err
	}
	asJSON := processAsJSONFlagValue(cmd)
	cmd.SilenceUsage = true
	ps, err := kanx.ListProcesses(cmd.Context(), addr)
	if err != nil {
		return err
	}
	for _, p := range ps {
		if asJSON {
			buf, err := protojson.Marshal(p)
			if err != nil {
				return err
			}
			fmt.Fprintln(out, string(buf))
		} else {
			fmt.Fprintln(out, "Process: ", p)
		}
	}
	return nil
}
