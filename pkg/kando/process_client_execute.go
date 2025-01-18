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
	"fmt"
	"io"

	"github.com/kanisterio/errkit"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/kanisterio/kanister/pkg/kanx"
)

func newProcessClientExecuteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "execute CMD ARG...",
		Short: "execute a new managed process",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runProcessClientExecute,
	}
	procesSignalProxyAddFlag(cmd)
	return cmd
}

func runProcessClientExecute(cmd *cobra.Command, args []string) error {
	return runProcessClientExecuteWithOutput(cmd.OutOrStdout(), cmd.ErrOrStderr(), cmd, args)
}

func runProcessClientExecuteWithOutput(stdout, stderr io.Writer, cmd *cobra.Command, args []string) error {
	addr, err := processAddressFlagValue(cmd)
	if err != nil {
		return err
	}
	proxy, err := processSignalProxyFlagValue(cmd)
	if err != nil {
		return err
	}
	asJSON := processAsJSONFlagValue(cmd)
	cmd.SilenceUsage = true
	ctx, canfn := context.WithCancel(cmd.Context())
	defer canfn()
	p, err := kanx.CreateProcess(ctx, addr, args[0], args[1:])
	if err != nil {
		return err
	}
	if asJSON {
		buf, err := protojson.Marshal(p)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, string(buf))
	} else {
		fmt.Fprintln(stdout, "Process: ", p)
	}

	pid := p.Pid
	errc := make(chan error)
	if proxy {
		proxySetup(ctx, addr, pid)
	}
	go func() { errc <- kanx.Stdout(ctx, addr, pid, stdout) }()
	go func() { errc <- kanx.Stderr(ctx, addr, pid, stderr) }()
	for i := 0; i < 2; i++ {
		err0 := <-errc
		// workaround bug in errkit
		if err0 != nil {
			err = errkit.Append(err, err0)
		}
	}
	return err
}
