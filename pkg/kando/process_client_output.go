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
	"os/signal"
	"strconv"
	"syscall"

	"github.com/kanisterio/errkit"
	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/kanx"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	processSignalProxyFlagName = "signal-proxy"
)

func newProcessClientOutputCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "output PID",
		Short: "stream output of a managed process",
		Args:  cobra.ExactArgs(1),
		RunE:  runProcessClientOutput,
	}
	cmd.PersistentFlags().BoolP(processSignalProxyFlagName, "p", false, "pass signals from client to server")
	return cmd
}

func processSignalProxyFlagValue(cmd *cobra.Command) (bool, error) {
	return cmd.Flags().GetBool(processSignalProxyFlagName)
}

func runProcessClientOutput(cmd *cobra.Command, args []string) error {
	return runProcessClientOutputWithOutput(cmd.OutOrStdout(), cmd.ErrOrStderr(), cmd, args)
}

func proxySetup(ctx context.Context, addr string, pid int64) {
	log.Info().WithContext(ctx).Print(fmt.Sprintf("signal proxy is running for process %d", pid))
	signalTermChan := make(chan os.Signal, 1)
	signal.Notify(signalTermChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
	BREAK:
		for {
			select {
			case sig := <-signalTermChan:
				ossig, ok := sig.(syscall.Signal)
				if !ok {
					log.Info().WithContext(ctx).Print(fmt.Sprintf("signal %v is invalid, ignored for process %d", sig, pid))
					continue
				}
				log.Info().WithContext(ctx).Print(fmt.Sprintf("signal %v received for process %d", sig, pid))
				_, err := kanx.SignalProcess(ctx, addr, pid, int64(ossig))
				if err != nil {
					signal.Reset(ossig)
					log.Error().WithContext(ctx).WithError(err).Print(fmt.Sprintf("error on signal %v for process %d", sig, pid))
					break BREAK
				}
				log.Info().WithContext(ctx).Print(fmt.Sprintf("signal %v sent for process %d", sig, pid))
			case <-ctx.Done():
				break BREAK
			}
		}
	}()
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
	proxy, err := processSignalProxyFlagValue(cmd)
	if err != nil {
		return err
	}
	ctx, canfn := context.WithCancel(cmd.Context())
	errc := make(chan error)
	if proxy {
		proxySetup(ctx, addr, pid)
	}
	cmd.SilenceUsage = true
	go func() { errc <- kanx.Stdout(ctx, addr, pid, stdout) }()
	go func() { errc <- kanx.Stderr(ctx, addr, pid, stderr) }()
	for i := 0; i < 2; i++ {
		err0 := <-errc
		if err0 != nil {
			err = errkit.Append(err, err0)
		}
	}
	canfn()
	return err
}
