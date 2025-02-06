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
	"os"
	"os/signal"
	"syscall"

	"github.com/kanisterio/errkit"
	"github.com/spf13/cobra"

	"github.com/kanisterio/kanister/pkg/kanx"
	"github.com/kanisterio/kanister/pkg/log"
)

const (
	processSignalProxyFlagName = "signal-proxy"
	processAsJSONFlagName      = "as-json"
	processAsQuietFlagName     = "quiet"
)

func newProcessClientCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client <command>",
		Short: "Send commands to the process server",
	}
	cmd.AddCommand(newProcessClientCreateCommand())
	cmd.AddCommand(newProcessClientExecuteCommand())
	cmd.AddCommand(newProcessClientGetCommand())
	cmd.AddCommand(newProcessClientListCommand())
	cmd.AddCommand(newProcessClientSignalCommand())
	cmd.AddCommand(newProcessClientOutputCommand())
	cmd.PersistentFlags().BoolP(processAsJSONFlagName, "j", false, "Display output as json")
	cmd.PersistentFlags().BoolP(processAsQuietFlagName, "q", false, "Quiet process information output")
	return cmd
}

func processAsJSONFlagValue(cmd *cobra.Command) bool {
	b, err := cmd.Flags().GetBool(processAsJSONFlagName)
	if err != nil {
		panic(err.Error())
	}
	return b
}

func processAsQuietFlagValue(cmd *cobra.Command) bool {
	b, err := cmd.Flags().GetBool(processAsQuietFlagName)
	if err != nil {
		panic(err.Error())
	}
	return b
}

func proxySetup(ctx context.Context, addr string, pid int64) {
	log.Info().WithContext(ctx).Print(fmt.Sprintf("signal proxy is running for process %d", pid))
	signalTermChan := make(chan os.Signal, 1)
	signal.Notify(signalTermChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		err := proxySignals(ctx, signalTermChan, addr, pid)
		if err != nil && !errkit.Is(err, context.Canceled) && !errkit.Is(err, context.DeadlineExceeded) {
			log.Info().WithContext(ctx).WithError(err)
		}
	}()
}

func proxySignals(ctx context.Context, signalTermChan chan os.Signal, addr string, pid int64) error {
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
				return errkit.New(fmt.Sprintf("error on signal %v for process %d", sig, pid))
			}
			log.Info().WithContext(ctx).Print(fmt.Sprintf("signal %v sent for process %d", sig, pid))
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func procesSignalProxyAddFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolP(processSignalProxyFlagName, "p", false, "pass signals from client to server")
}

func processSignalProxyFlagValue(cmd *cobra.Command) (bool, error) {
	return cmd.Flags().GetBool(processSignalProxyFlagName)
}
