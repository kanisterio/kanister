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
	"io"
	"strconv"

	"github.com/kanisterio/errkit"
	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/kanx"
)

func newProcessClientOutputCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "output PID",
		Short: "stream output of a managed process",
		Args:  cobra.ExactArgs(1),
		RunE:  runProcessClientOutput,
	}
	processSignalProxyAddFlag(cmd)
	processExitProxyAddFlag(cmd)
	return cmd
}

func runProcessClientOutput(cmd *cobra.Command, args []string) error {
	return runProcessClientOutputWithOutput(cmd.OutOrStdout(), cmd.ErrOrStderr(), cmd, args)
}

func runProcessClientOutputWithOutput(stdout, stderr io.Writer, cmd *cobra.Command, args []string) error {
	pid, err := strconv.ParseInt(args[0], 0, 64)
	if err != nil {
		return err
	}
	addr, err := processAddressFlagValue(cmd)
	if err != nil {
		return err
	}
	signalProxy, err := processSignalProxyFlagValue(cmd)
	if err != nil {
		return err
	}
	exitProxy, err := processExitProxyFlagValue(cmd)
	if err != nil {
		return err
	}
	asQuiet := processAsQuietFlagValue(cmd)
	if asQuiet {
		cmd.SilenceErrors = true
	}
	cmd.SilenceUsage = true
	ctx, canfn := context.WithCancel(cmd.Context())
	defer canfn()
	errc := make(chan error)
	if signalProxy {
		proxySetup(ctx, addr, pid)
	}
	go func() { errc <- kanx.Stdout(ctx, addr, pid, stdout) }()
	go func() { errc <- kanx.Stderr(ctx, addr, pid, stderr) }()
	for i := 0; i < 2; i++ {
		err0 := <-errc
		if err0 != nil {
			// workaround bug in errkit
			err = errkit.Append(err, err0)
		}
	}
	close(errc)
	if err != nil {
		return err
	}
	if !exitProxy {
		return nil
	}
	return err
}
