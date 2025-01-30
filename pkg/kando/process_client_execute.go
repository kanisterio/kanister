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
	"os"

	"github.com/kanisterio/errkit"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/kanisterio/kanister/pkg/kanx"
)

var (
	exit = os.Exit
)

const (
	GrpcCodeOffset = 15
)

func newProcessClientExecuteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "execute CMD ARG...",
		Short: "execute a new managed process and follow output. provides option of forwarding signals from client to server",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runProcessClientExecute,
	}
	procesSignalProxyAddFlag(cmd)
	return cmd
}

func runProcessClientExecute(cmd *cobra.Command, args []string) error {
	err := runProcessClientExecuteWithOutput(cmd.OutOrStdout(), cmd.ErrOrStderr(), cmd, args)
	if err == nil {
		return nil
	}
	// err is a positive command exit code
	err0, ok := err.(kanx.ProcessExitCode)
	if ok {
		exit(int(err0))
	}
	// err is gRPC error.  this will tell users of connectivity problems
	// with the server
	err1, ok := status.FromError(err)
	if ok && err1.Code() != codes.OK {
		exit(int(GrpcCodeOffset + err1.Code()))
	}
	return err
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
	asQuiet := processAsQuietFlagValue(cmd)
	cmd.SilenceUsage = true
	ctx, canfn0 := context.WithCancel(cmd.Context())
	defer canfn0()
	// start the process in the server
	p, err := kanx.CreateProcess(ctx, addr, args[0], args[1:])
	if err != nil {
		return err
	}
	// output the process metadata
	if !asQuiet {
		if asJSON {
			buf, err := protojson.Marshal(p)
			if err != nil {
				return err
			}
			fmt.Fprintln(stdout, string(buf))
		} else {
			fmt.Fprintln(stdout, "Process: ", p)
		}
	}
	pid := p.Pid
	// setup signal proxies if requested
	errc := make(chan error)
	if proxy {
		proxySetup(ctx, addr, pid)
	}
	// process stdout and stderr in background
	go func() { errc <- kanx.Stdout(ctx, addr, pid, stdout) }()
	go func() { errc <- kanx.Stderr(ctx, addr, pid, stderr) }()
	// wait for process completion keeping errors for Stdout and Stderr calls
	for i := 0; i < 2; i++ {
		err0 := <-errc
		// workaround bug in errkit
		if err0 != nil {
			err = errkit.Append(err, err0)
		}
	}
	close(errc)
	if err != nil {
		return err
	}
	// get terminal state of the process.  Ideally this would be returned in Stdout,
	// Stderr or via a Wait function for now we wait for process completion and then
	// call GetProcess to get the final state
	ctx, canfn1 := context.WithCancel(cmd.Context())
	defer canfn1()
	p, err = kanx.GetProcess(ctx, addr, pid)
	if err != nil {
		return err
	}
	// exit codes from process need to be proxied so that KanX clients can respond
	// to them
	if p.ExitCode != 0 {
		err = kanx.ProcessExitCode(p.ExitCode)
	}
	return err
}
