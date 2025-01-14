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
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/kanisterio/kanister/pkg/kanx"
)

func newProcessClientSignalCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signal SIGNAL PID",
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
	pid, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}
	signal, err := strconv.Atoi(args[1])
	if err != nil {
		return err
	}
	addr, err := processAddressFlagValue(cmd)
	if err != nil {
		return err
	}
	asJSON := processAsJSONFlagValue(cmd)
	cmd.SilenceUsage = true
	p, err := kanx.SignalProcess(cmd.Context(), addr, int64(pid), int32(signal))
	if err != nil {
		return err
	}
	if asJSON {
		buf, err := protojson.Marshal(p)
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(buf))
	} else {
		fmt.Fprintln(out, "Process: ", p.String())
	}
	return nil
}
