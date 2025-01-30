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
	"strconv"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/kanisterio/kanister/pkg/kanx"
)

func newProcessClientSignalCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signal PID SIGNAL",
		Short: "send a signal to a managed process",
		Args:  cobra.ExactArgs(2),
		RunE:  runProcessClientSignal,
	}
	return cmd
}

func runProcessClientSignal(cmd *cobra.Command, args []string) error {
	return runProcessClientSignalWithOutput(cmd.OutOrStdout(), cmd, args)
}

func runProcessClientSignalWithOutput(out io.Writer, cmd *cobra.Command, args []string) error {
	pid, err := strconv.ParseInt(args[0], 0, 64)
	if err != nil {
		return err
	}
	signal, err := strconv.ParseInt(args[1], 0, 64)
	if err != nil {
		return err
	}
	addr, err := processAddressFlagValue(cmd)
	if err != nil {
		return err
	}
	asQuiet := processAsQuietFlagValue(cmd)
	asJSON := processAsJSONFlagValue(cmd)
	cmd.SilenceUsage = true
	p, err := kanx.SignalProcess(cmd.Context(), addr, pid, signal)
	if err != nil {
		return err
	}
	if asQuiet {
		return nil
	}
	if asJSON {
		buf, err := protojson.Marshal(p)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(buf))
	} else {
		fmt.Fprintln(out, "Process: ", p)
	}
	return nil
}
